// Copyright 2014-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package eventhandler

import (
	"sync"

	"github.com/aws/amazon-ecs-agent/agent/api"
)

// a state change that may have a container and, optionally, a task event to
// send
type sendableEvent struct {
	// Either is a contaienr event or a task event
	isContainerEvent bool

	containerSent   bool
	containerChange api.ContainerStateChange

	taskSent   bool
	taskChange api.TaskStateChange

	lock sync.RWMutex
}

func (event *sendableEvent) String() string {
	event.lock.RLock()
	defer event.lock.RUnlock()

	if event.isContainerEvent {
		return "ContainerChange: " + event.containerChange.String()
	} else {
		return "TaskChange: " + event.taskChange.String()
	}
}

func newSendableContainerEvent(event api.ContainerStateChange) *sendableEvent {
	return &sendableEvent{
		isContainerEvent: true,
		containerSent:    false,
		containerChange:  event,
	}
}

func newSendableTaskEvent(event api.TaskStateChange) *sendableEvent {
	return &sendableEvent{
		isContainerEvent: false,
		taskSent:         false,
		taskChange:       event,
	}
}

func (event *sendableEvent) taskArn() string {
	if event.isContainerEvent {
		return event.containerChange.TaskArn
	}
	return event.taskChange.TaskARN
}

func (event *sendableEvent) taskShouldBeSent() bool {
	event.lock.RLock()
	defer event.lock.RUnlock()
	if event.isContainerEvent {
		return false
	}
	tevent := event.taskChange
	if tevent.Status == api.TaskStatusNone {
		return false // defensive programming :)
	}
	if event.taskSent || (tevent.Task != nil && tevent.Task.GetSentStatus() >= tevent.Status) {
		return false // redundant event
	}
	return true
}

func (event *sendableEvent) taskAttachmentShouldBeSent() bool {
	event.lock.RLock()
	defer event.lock.RUnlock()
	if event.isContainerEvent {
		return false
	}
	tevent := event.taskChange
	return tevent.Status == api.TaskStatusNone && // Task Status is not set for attachments as task record has yet to be streamed down
		tevent.Attachment != nil && // Task has attachment records
		!tevent.Attachment.IsSent() // Task status hasn't already been sent
}

func (event *sendableEvent) containerShouldBeSent() bool {
	event.lock.RLock()
	defer event.lock.RUnlock()
	if !event.isContainerEvent {
		return false
	}
	cevent := event.containerChange
	if event.containerSent || (cevent.Container != nil && cevent.Container.GetSentStatus() >= cevent.Status) {
		return false
	}
	return true
}

func (event *sendableEvent) setSent() {
	event.lock.Lock()
	defer event.lock.Unlock()
	if event.isContainerEvent {
		event.containerSent = true
	} else {
		event.taskSent = true
	}
}
