/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package simplebft

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/op/go-logging"
)

var testLog = logging.MustGetLogger("test")

func init() {
	logging.SetLevel(logging.NOTICE, "")
	logging.SetLevel(logging.NOTICE, "test")
	logging.SetLevel(logging.DEBUG, "sbft")
}

func TestSBFT(t *testing.T) {
	N := uint64(4)
	sys := newTestSystem(N)
	var repls []*SBFT
	var adapters []*testSystemAdapter
	for i := uint64(0); i < N; i++ {
		a := sys.NewAdapter(i)
		s, err := New(i, &Config{N: N, F: 1, BatchDurationNsec: 2000000000, BatchSizeBytes: 10, RequestTimeoutNsec: 20000000000}, a)
		if err != nil {
			t.Fatal(err)
		}
		repls = append(repls, s)
		adapters = append(adapters, a)
	}
	r1 := []byte{1, 2, 3}
	repls[0].Request(r1)
	sys.Run()
	r2 := []byte{3, 1, 2}
	r3 := []byte{3, 5, 2}
	repls[1].Request(r2)
	repls[1].Request(r3)
	sys.Run()
	for _, a := range adapters {
		if len(a.batches) != 2 {
			t.Fatal("expected execution of 2 batches")
		}
		if !reflect.DeepEqual([][]byte{r1}, a.batches[0]) {
			t.Error("wrong request executed (1)")
		}
		if !reflect.DeepEqual([][]byte{r2, r3}, a.batches[1]) {
			t.Error("wrong request executed (2)")
		}
	}
}

func TestN1(t *testing.T) {
	N := uint64(1)
	sys := newTestSystem(N)
	var repls []*SBFT
	var adapters []*testSystemAdapter
	for i := uint64(0); i < N; i++ {
		a := sys.NewAdapter(i)
		s, err := New(i, &Config{N: N, F: 0, BatchDurationNsec: 2000000000, BatchSizeBytes: 10, RequestTimeoutNsec: 20000000000}, a)
		if err != nil {
			t.Fatal(err)
		}
		repls = append(repls, s)
		adapters = append(adapters, a)
	}
	r1 := []byte{1, 2, 3}
	repls[0].Request(r1)
	sys.Run()
	for _, a := range adapters {
		if len(a.batches) != 1 {
			t.Fatal("expected execution of 1 batch")
		}
		if !reflect.DeepEqual([][]byte{r1}, a.batches[0]) {
			t.Error("wrong request executed (1)")
		}
	}
}

func TestByzPrimary(t *testing.T) {
	N := uint64(4)
	sys := newTestSystem(N)
	var repls []*SBFT
	var adapters []*testSystemAdapter
	for i := uint64(0); i < N; i++ {
		a := sys.NewAdapter(i)
		s, err := New(i, &Config{N: N, F: 1, BatchDurationNsec: 2000000000, BatchSizeBytes: 1, RequestTimeoutNsec: 20000000000}, a)
		if err != nil {
			t.Fatal(err)
		}
		repls = append(repls, s)
		adapters = append(adapters, a)
	}

	r1 := []byte{1, 2, 3}
	r2 := []byte{5, 6, 7}

	// change preprepare to 2, 3
	sys.filterFn = func(e testElem) (testElem, bool) {
		if msg, ok := e.ev.(*testMsgEvent); ok {
			if pp := msg.msg.GetPreprepare(); pp != nil && msg.src == 0 && msg.dst >= 2 {
				d := &DigestSet{}
				proto.Unmarshal(pp.Set, d)
				d.Digest[0] = r2
				pp := *pp
				pp.Set, _ = proto.Marshal(d)
				msg.msg = &Msg{&Msg_Preprepare{&pp}}
			}
		}
		return e, true
	}

	repls[0].Request(r1)
	sys.Run()
	for _, a := range adapters {
		if len(a.batches) != 1 {
			t.Fatal("expected execution of 1 batch")
		}
		if !reflect.DeepEqual([][]byte{r2}, a.batches[0]) {
			t.Error("wrong request executed")
		}
	}
}

func TestViewChange(t *testing.T) {
	N := uint64(4)
	sys := newTestSystem(N)
	var repls []*SBFT
	var adapters []*testSystemAdapter
	for i := uint64(0); i < N; i++ {
		a := sys.NewAdapter(i)
		s, err := New(i, &Config{N: N, F: 1, BatchDurationNsec: 2000000000, BatchSizeBytes: 1, RequestTimeoutNsec: 20000000000}, a)
		if err != nil {
			t.Fatal(err)
		}
		repls = append(repls, s)
		adapters = append(adapters, a)
	}

	// network outage after prepares are received
	sys.filterFn = func(e testElem) (testElem, bool) {
		if msg, ok := e.ev.(*testMsgEvent); ok {
			if c := msg.msg.GetCommit(); c != nil && c.Seq.View == 0 {
				return e, false
			}
		}
		return e, true
	}

	r1 := []byte{1, 2, 3}
	repls[0].Request(r1)
	sys.Run()
	for _, a := range adapters {
		if len(a.batches) != 1 {
			t.Fatal("expected execution of 1 batch")
		}
		if !reflect.DeepEqual([][]byte{r1}, a.batches[0]) {
			t.Error("wrong request executed (1)")
		}
	}
}