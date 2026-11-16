package st

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"reflect"
	"testing"
	"time"
)

func TestGUIPipeExchangeAndCancel(t *testing.T) {
	var key [16]byte
	if _, err := rand.Read(key[:]); err != nil {
		t.Fatal(err)
	}
	id := hex.EncodeToString(key[:])
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ready := make(chan error, 1)
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		RunGUIPipe(ctx, id, func(req GUIRequest) GUIResponse {
			return GUIResponse{Path: req.BasePath, Nodes: *req.NodeIDs}
		}, ready)
	}()
	if err := <-ready; err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		ids := []int{i}
		reply, err := ExchangeGUI(ctx, id, GUIRequest{Action: "status", BasePath: "model.inp", NodeIDs: &ids})
		if err != nil {
			t.Fatal(err)
		}
		if reply.Path != "model.inp" || !reflect.DeepEqual(reply.Nodes, ids) {
			t.Fatal(reply)
		}
	}
	cancel()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("named pipe did not close")
	}
}
