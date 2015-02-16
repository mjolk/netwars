// +build !appengine

package testutils

import (
	"appengine"
	"appengine/taskqueue"
	"testing"
)

func CheckQueue(c appengine.Context, t *testing.T, nrTasks int) {
	stats, err := taskqueue.QueueStats(c, []string{"default"}, 0) // fetch all of them
	if err != nil {
		t.Fatalf("Could not get taskqueue statistics")
	}
	t.Logf("TaskStatistics = %#v", stats)
	if len(stats) == 0 {
		t.Fatalf("Queue statistics are empty")
	} else if stats[0].Tasks != nrTasks {
		t.Fatalf("Could not find the task we just added")
	}
}

func PurgeQueue(c appengine.Context, t *testing.T) {
	err := taskqueue.Purge(c, "default")
	if err != nil {
		t.Fatalf("Could not purge the queue")
	}
}
