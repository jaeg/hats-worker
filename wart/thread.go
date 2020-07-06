package wart

import (
	"strconv"
	"time"

	"github.com/robertkrimen/otto"
	log "github.com/sirupsen/logrus"
)

type ThreadMeta struct {
	Key     string
	Stopped bool
	vm      *otto.Otto
}

func (tm *ThreadMeta) getStatus(w *Wart) (status string) {
	status = w.Client.HGet(ctx, tm.Key, "Status").Val()
	return
}

func (tm *ThreadMeta) getState(w *Wart) (state string) {
	state = w.Client.HGet(ctx, tm.Key, "State").Val()
	return
}

func (tm *ThreadMeta) getSource(w *Wart) (source string) {
	source = w.Client.HGet(ctx, tm.Key, "Source").Val()
	return
}

func (tm *ThreadMeta) getHeartBeat(w *Wart) (hb int, err error) {
	hbString := w.Client.HGet(ctx, tm.Key, "Heartbeat").Val()
	hb, err = strconv.Atoi(hbString)

	return
}

func (tm *ThreadMeta) getOwner(w *Wart) (owner string) {
	owner = w.Client.HGet(ctx, tm.Key, "Owner").Val()

	return
}

func (tm *ThreadMeta) getDeadSeconds(w *Wart) (deadSeconds int, err error) {
	deadSeconds, err = w.Client.HGet(ctx, tm.Key, "DeadSeconds").Int()
	return
}

func (tm *ThreadMeta) take(w *Wart) {
	log.Info("Taking thread ", tm.Key)
	tm.Stopped = false
	w.Client.HSet(ctx, tm.Key, "State", RUNNING)
	w.Client.HSet(ctx, tm.Key, "Heartbeat", time.Now().UnixNano())
	w.Client.HSet(ctx, tm.Key, "Owner", w.WartName)
	go tm.run(w)
}

func (tm *ThreadMeta) stop(w *Wart) {
	if tm.getOwner(w) == w.WartName && !tm.Stopped {
		log.Info("Stopping thread ", tm.Key)
		tm.Stopped = true
		w.Client.HSet(ctx, tm.Key, "State", STOPPED)
		if tm.vm != nil {
			tm.vm.Interrupt <- func() {
				log.Error("Stopping threads")
				return
			}
		}
	}
}

func (tm *ThreadMeta) run(w *Wart) {
	log.Info("Starting Thread ", tm.Key)

	tm.vm = otto.New()
	tm.vm.Interrupt = make(chan func(), 1)
	applyLibrary(w, tm.vm)
	source := tm.getSource(w)
	if source == "" {
		log.Error("Source empty for thread ", tm.Key)
		return
	}
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

	hang, hangErr := w.Client.HGet(ctx, tm.Key, "Hang").Int()
	if hangErr == nil {
		// Check to make sure since should stop could of changed.
		if !tm.Stopped {
			_, err := tm.vm.Run("if (typeof init === 'function') {init()}")
			if err != nil {
				w.Client.HSet(ctx, tm.Key, "State", CRASHED)
				w.Client.HSet(ctx, tm.Key, "Status", DISABLED)
				w.Client.HSet(ctx, tm.Key, "Error", err.Error())
				w.Client.HSet(ctx, tm.Key, "ErrorTime", time.Now())
				log.WithError(err).Error("Error running init() in script " + tm.Key)
				return
			}

			time.Sleep(time.Duration(hang))
		}

		for w.Healthy && !tm.Stopped {
			w.Client.HSet(ctx, tm.Key, "Heartbeat", time.Now().UnixNano())

			//Get status and stop if disabled.
			status := tm.getStatus(w)
			owner := tm.getOwner(w)
			//If script has been disabled don't run it.
			if status == DISABLED {
				log.Warn(tm.Key, "Was disabled.  Stopping thread.")
				w.Client.HSet(ctx, tm.Key, "State", STOPPED)
				tm.Stopped = true
				continue
			}

			//If we aren't the owner anymore don't run it.
			if owner != w.WartName {
				tm.Stopped = true
				continue
			}

			// Check to make sure since should stop could of changed.
			if !tm.Stopped {
				_, err := tm.vm.Run("if (typeof main === 'function') {main()}")
				if err != nil {
					w.Client.HSet(ctx, tm.Key, "State", CRASHED)
					w.Client.HSet(ctx, tm.Key, "Status", DISABLED)
					w.Client.HSet(ctx, tm.Key, "Error", err.Error())
					w.Client.HSet(ctx, tm.Key, "ErrorTime", time.Now())
					log.WithError(err).Error("Error running main() in script " + tm.Key)
					return
				}

				time.Sleep(time.Duration(hang))
			}
		}

		//Thread has ended, run any cleanup there might be.
		_, err := tm.vm.Run("if (typeof init === 'function') {cleanup()}")
		if err != nil {
			log.WithError(err).Error("Error cleaning up thread: ", tm.Key)
		}

	} else {
		log.WithError(hangErr).Error("Error hanging")
	}
	w.Client.HSet(ctx, tm.Key, "State", STOPPED)
}
