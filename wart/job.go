package wart

import (
	"strconv"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

//JobMeta Struct that represents a job.
type JobMeta struct {
	Key        string
	Stopped    bool
	vm         *otto.Otto
	cron       *cron.Cron
	cronString string
}

func (jm *JobMeta) getVM() *otto.Otto {
	return jm.vm
}

func (jm *JobMeta) getStatus(w *Wart) (status string) {
	status = w.Client.HGet(ctx, jm.Key, "Status").Val()
	return
}

func (jm *JobMeta) getCron(w *Wart) (status string) {
	status = w.Client.HGet(ctx, jm.Key, "Cron").Val()
	return
}

func (jm *JobMeta) getState(w *Wart) (state string) {
	state = w.Client.HGet(ctx, jm.Key, "State").Val()
	return
}

func (jm *JobMeta) getSource(w *Wart) (source string) {
	source = w.Client.HGet(ctx, jm.Key, "Source").Val()
	return
}

func (jm *JobMeta) getHeartBeat(w *Wart) (hb int, err error) {
	hbString := w.Client.HGet(ctx, jm.Key, "Heartbeat").Val()
	hb, err = strconv.Atoi(hbString)

	return
}

func (jm *JobMeta) getOwner(w *Wart) (owner string) {
	owner = w.Client.HGet(ctx, jm.Key, "Owner").Val()

	return
}

func (jm *JobMeta) schedule(w *Wart) {
	if jm.cron == nil || jm.cronString != jm.getCron(w) {
		log.Info("Setting up job cron for ", jm.Key, " cron: ", jm.getCron(w))
		jm.cron = newWithSeconds()
		jm.cron.Start()
		jm.cron.AddFunc(jm.getCron(w), func() {
			go jm.run(w)
		})
		jm.cronString = jm.getCron(w)
	}
}

func (jm *JobMeta) disable(w *Wart) {
	if jm.getOwner(w) == w.WartName && !jm.Stopped {
		log.Info("Disabling thread ", jm.Key)
		jm.Stopped = true
		w.Client.HSet(ctx, jm.Key, "State", STOPPED)
		w.Client.HSet(ctx, jm.Key, "Status", DISABLED)
		if jm.vm != nil {
			jm.vm.Interrupt <- func() {
				log.Error("Disabled thread")
				return
			}
		}
	}
}

func (jm *JobMeta) run(w *Wart) {
	log.Info("Starting job ", jm.Key)
	if jm.getStatus(w) == DISABLED {
		jm.cron.Stop()
		jm.cron = nil
		log.Info("Job disabled ", jm.Key)
	}
	if jm.getOwner(w) == "" {
		jm.Stopped = false
		w.Client.HSet(ctx, jm.Key, "State", RUNNING)
		w.Client.HSet(ctx, jm.Key, "Heartbeat", time.Now().UnixNano())
		w.Client.HSet(ctx, jm.Key, "Owner", w.WartName)

		jm.vm = otto.New()
		jm.vm.Interrupt = make(chan func(), 1)
		applyLibrary(w, jm)
		source := jm.getSource(w)
		if source == "" {
			log.Error("Source empty for thread ", jm.Key)
			return
		}

		//Check one last time to make sure someone didn't beat us.
		if jm.getOwner(w) == w.WartName {
			//Get whole script in memory.
			_, err := jm.vm.Run(source)
			if err != nil {
				w.Client.HSet(ctx, jm.Key, "State", CRASHED)
				w.Client.HSet(ctx, jm.Key, "Status", DISABLED)
				w.Client.HSet(ctx, jm.Key, "Error", err.Error())
				w.Client.HSet(ctx, jm.Key, "ErrorTime", time.Now())
				log.WithError(err).Error("Syntax error in script.")
				return
			}

			w.Client.HSet(ctx, jm.Key, "State", STOPPED)
			w.Client.HSet(ctx, jm.Key, "Owner", "")
		}
	}
}
