// Copyright 2020-present Yarn.social
// SPDX-License-Identifier: AGPL-3.0-or-later

package internal

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"go.mills.io/tasks"
)

// AudioTask is a task to transcode an audio file
type AudioTask struct {
	*tasks.BaseTask

	conf *Config
	fn   string
}

// NewAudioTask returns a new AudioTask instance.
//
// The returned task is a Go routine safe object, but it's not safe to access
// the underlying fields directly. Instead, use the methods and functions
// provided by the tasks package and this package for interacting with the
// task.
func NewAudioTask(conf *Config, fn string) *AudioTask {
	return &AudioTask{
		BaseTask: tasks.NewBaseTask(),

		conf: conf,
		fn:   fn,
	}
}

// String returns a string representation of the AudioTask, including its type and ID.
func (t *AudioTask) String() string {
	return fmt.Sprintf("%T: %s", t, t.ID())
}

// Run executes the audio transcoding task. It sets the task state to running,
// logs the start of the process, and defines audio options for transcoding.
// The function calls TranscodeAudio to perform the transcoding operation.
// If an error occurs, it logs the error and marks the task as failed.
// On success, it logs the completion and stores the resulting media URI.
func (t *AudioTask) Run() error {
	defer t.Done()
	t.SetState(tasks.TaskStateRunning)

	log.Infof("starting audio transcode task for %s", t.fn)

	opts := &AudioOptions{
		Resample:   true,
		Channels:   1,
		Samplerate: 16000,
		Bitrate:    96,
	}
	mediaURI, err := TranscodeAudio(t.conf, t.fn, mediaDir, "", opts)
	if err != nil {
		log.WithError(err).Errorf("error transcoding audio %s", t.fn)
		return t.Fail(err)
	}
	log.Infof("audio transcode complete for %s with uri %s", t.fn, mediaURI)

	t.SetData("mediaURI", mediaURI)

	return nil
}
