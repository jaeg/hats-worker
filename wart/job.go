package wart

import (
	"strconv"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type JobMeta struct {
	Key        string
	Stopped    bool
	vm         *otto.Otto
	cron       *cron.Cron
	cronString string
}

func (tm *JobMeta) getStatus(w *Wart) (status string) {
	status = w.Client.HGet(ctx, tm.Key, "Status").Val()
	return
}

func (tm *JobMeta) getCron(w *Wart) (status string) {
	status = w.Client.HGet(ctx, tm.Key, "Cron").Val()
	return
}

func (tm *JobMeta) getState(w *Wart) (state string) {
	state = w.Client.HGet(ctx, tm.Key, "State").Val()
	return
}

func (tm *JobMeta) getSource(w *Wart) (source string) {
	source = w.Client.HGet(ctx, tm.Key, "Source").Val()
	return
}

func (tm *JobMeta) getHeartBeat(w *Wart) (hb int, err error) {
	hbString := w.Client.HGet(ctx, tm.Key, "Heartbeat").Val()
	hb, err = strconv.Atoi(hbString)

	return
}

func (tm *JobMeta) getOwner(w *Wart) (owner string) {
	owner = w.Client.HGet(ctx, tm.Key, "Owner").Val()

	return
}

func (tm *JobMeta) getDeadSeconds(w *Wart) (deadSeconds int, err error) {
	deadSeconds, err = w.Client.HGet(ctx, tm.Key, "DeadSeconds").Int()
	return
}
func (tm *JobMeta) schedule(w *Wart) {
	if tm.cron == nil || tm.cronString != tm.getCron(w) {
		log.Info("Setting up job cron for ", tm.Key, " cron: ", tm.getCron(w))
		tm.cron = newWithSeconds()
		tm.cron.Start()
		tm.cron.AddFunc(tm.getCron(w), func() {
			go tm.run(w)
		})
		tm.cronString = tm.getCron(w)
	}
}

func (tm *JobMeta) stop(w *Wart) {
	if tm.getOwner(w) == w.WartName && !tm.Stopped {
		log.Info("Stopping thread ", tm.Key)
		tm.Stopped = true
		tm.cron.Stop()
		tm.cron = nil
		w.Client.HSet(ctx, tm.Key, "State", STOPPED)
		if tm.vm != nil {
			tm.vm.Interrupt <- func() {
				log.Error("Stopping threads")
				return
			}
		}
	}
}

func (tm *JobMeta) disable(w *Wart) {
	if tm.getOwner(w) == w.WartName && !tm.Stopped {
		log.Info("Disabling thread ", tm.Key)
		tm.Stopped = true
		w.Client.HSet(ctx, tm.Key, "State", STOPPED)
		w.Client.HSet(ctx, tm.Key, "Status", DISABLED)
		if tm.vm != nil {
			tm.vm.Interrupt <- func() {
				log.Error("Disabled thread")
				return
			}
		}
	}
}

func (tm *JobMeta) run(w *Wart) {
	log.Info("Starting job ", tm.Key)
	if tm.getStatus(w) == DISABLED {
		tm.cron.Stop()
		tm.cron = nil
		log.Info("Job disabled ", tm.Key)
	}
	if tm.getOwner(w) == "" {
		//RUN HERE
		tm.Stopped = false
		w.Client.HSet(ctx, tm.Key, "State", RUNNING)
		w.Client.HSet(ctx, tm.Key, "Heartbeat", time.Now().UnixNano())
		w.Client.HSet(ctx, tm.Key, "Owner", w.WartName)

		tm.vm = otto.New()
		tm.vm.Interrupt = make(chan func(), 1)
		applyLibraryJob(w, tm)
		source := tm.getSource(w)
		if source == "" {
			log.Error("Source empty for thread ", tm.Key)
			return
		}

		//Check one last time to make sure someone didn't beat us.
		if tm.getOwner(w) == w.WartName {
			//Get whole script in memory.
			_, err := tm.vm.Run(source)
			if err != nil {
				w.Client.HSet(ctx, tm.Key, "State", CRASHED)
				w.Client.HSet(ctx, tm.Key, "Status", DISABLED)
				w.Client.HSet(ctx, tm.Key, "Error", err.Error())
				w.Client.HSet(ctx, tm.Key, "ErrorTime", time.Now())
				log.WithError(err).Error("Syntax error in script.")
				return
			}

			w.Client.HSet(ctx, tm.Key, "State", STOPPED)
			w.Client.HSet(ctx, tm.Key, "Owner", "")
		}
	}
}
