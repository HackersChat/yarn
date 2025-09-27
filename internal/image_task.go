// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"go.mills.io/tasks"
)

type ImageTask struct {
	*tasks.BaseTask

	conf *Config
	fn   string
}

func NewImageTask(conf *Config, fn string) *ImageTask {
	return &ImageTask{
		BaseTask: tasks.NewBaseTask(),

		conf: conf,
		fn:   fn,
	}
}

func (t *ImageTask) String() string { return fmt.Sprintf("%T: %s", t, t.ID()) }
func (t *ImageTask) Run() error {
	defer t.Done()
	t.SetState(tasks.TaskStateRunning)

	log.Infof("starting image processing task for %s", t.fn)

	opts := &ImageOptions{Resize: true, Width: t.conf.MediaResolution, Height: 0}
	mediaURI, err := ProcessImage(t.conf, t.fn, mediaDir, "", opts)
	if err != nil {
		log.WithError(err).Errorf("error processing image %s", t.fn)
		return t.Fail(err)
	}
	log.Infof("image processing complete for %s with uri %s", t.fn, mediaURI)

	t.SetData("mediaURI", mediaURI)

	return nil
}
