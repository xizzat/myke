package core

import (
	"github.com/apex/log"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"

	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/syntax"
	"context"
	"os/signal"
	"syscall"
)

// Execution represents a task and context being invoked
type Execution struct {
	Workspace *Workspace
	Query     *Query
	Project   *Project
	Task      *Task
	DryRun    bool
	Verbose   int
}

// ExecuteQuery executes the given query in the workspace
func ExecuteQuery(w *Workspace, q Query, dryRun bool, verbose int) error {
	matches := q.search(w)
	if len(matches) == 0 {
		return errors.New("no task matched: " + q.Raw)
	}
	for _, match := range matches {
		e := Execution{
			Workspace: w,
			Query:     &q,
			Project:   &match.Project,
			Task:      &match.Task,
			DryRun:    dryRun,
			Verbose:   verbose,
		}
		err := e.Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

// Execute executes the current task
func (e *Execution) Execute() error {
	start := time.Now()
	displayName := e.Project.Name + "/" + e.Task.Name
	if e.DryRun {
		log.Infof("%v: Will run", displayName)
		return nil
	}
	log.Infof("%v: Running", displayName)

	err := retry(func(attempt int) (bool, error) {
		err := e.executeTask()
		if err != nil && attempt < e.Task.Retry {
			log.Debugf("%v: Failed, Retrying %v/%v in %v", displayName, attempt, e.Task.Retry, e.Task.RetryDelay)
			time.Sleep(e.Task.RetryDelay)
		}
		return attempt < e.Task.Retry, err
	})

	elapsed := time.Since(start)
	if err != nil {
		log.Infof("%v: Failed, Took: %v", displayName, elapsed)
	} else {
		log.Infof("%v: Completed, Took: %v", displayName, elapsed)
	}
	return errors.Wrap(err, displayName)
}

func (e *Execution) executeTask() error {
	err := e.executeCmd(e.Task.Before)
	if err == nil {
		err = e.executeCmd(e.Task.Cmd)
		if err == nil {
			err = e.executeCmd(e.Task.After)
		}
	}

	if err != nil && len(e.Task.Error) > 0 {
		e.executeCmd(e.Task.Error)
	}
	return err
}

func (e *Execution) executeCmd(cmd string) error {
	if len(strings.TrimSpace(cmd)) == 0 {
		return nil
	}

	env, err := e.env()
	if err != nil {
		return err
	}

	cmd, err = RenderTemplate(cmd, env)
	if err != nil {
		return err
	}

	if e.Verbose > 0 {
		log.Infof("Executing cmd: \n```\n%v\n```", cmd)
	}

	p, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		return err
	}

	runnerEnv, err := interp.EnvFromList(mapToSlice(env))
	if err != nil {
		return err
	}

	r := interp.Runner{
		Context: getSignalContext(),
		Dir:     e.Project.Cwd,
		Env:     runnerEnv,

		Exec: interp.DefaultExec,
		Open: interp.OpenDevImpls(interp.DefaultOpen),

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err = r.Reset(); err != nil {
		return err
	}

	return r.Run(p)
}

func (e *Execution) env() (map[string]string, error) {
	myke, _ := osext.Executable()
	extraEnv := map[string]string{
		"myke":         myke,
		"MYKE_PROJECT": e.Project.Name,
		"MYKE_TASK":    e.Task.Name,
		"MYKE_CWD":     e.Project.Cwd,
	}
	env := mergeEnv(mergeEnv(e.Project.Env, extraEnv), OsEnv())
	env["PATH"] = strings.Join([]string{env["PATH"], os.Getenv("PATH")}, pathSep)
	env = union(env, e.Query.Params)

	env, err := RenderParams(env)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func getSignalContext() context.Context {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ch
		cancel()
	}()
	return ctx
}
