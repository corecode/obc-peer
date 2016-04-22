/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package obcpbft

import (
	"github.com/golang/protobuf/proto"
)

func (instance *pbftCore) persistQSet() {
	var qset []*ViewChange_PQ

	for _, q := range instance.calcQSet() {
		qset = append(qset, q)
	}

	instance.persistPQSet("qset", qset)
}

func (instance *pbftCore) persistPSet() {
	var pset []*ViewChange_PQ

	for _, p := range instance.calcPSet() {
		pset = append(pset, p)
	}

	instance.persistPQSet("pset", pset)
}

func (instance *pbftCore) persistPQSet(key string, set []*ViewChange_PQ) {
	raw, err := proto.Marshal(&PQset{set})
	if err != nil {
		logger.Warning("Replica %d could not persist pqset: %s", instance.id, err)
		return
	}
	instance.consumer.StoreState(key, raw)
}

func (instance *pbftCore) restorePQSet(key string) []*ViewChange_PQ {
	raw, err := instance.consumer.ReadState(key)
	if err != nil {
		logger.Debug("Replica %d could not restore state %s: %s", instance.id, key, err)
		return nil
	}
	val := &PQset{}
	err = proto.Unmarshal(raw, val)
	if err != nil {
		logger.Error("Replica %d could not unmarshal %s - local state is damaged: %s", instance.id, err)
		return nil
	}
	return val.GetSet()
}

func (instance *pbftCore) persistRequest(digest string) {
	req := instance.reqStore[digest]
	raw, err := proto.Marshal(req)
	if err != nil {
		logger.Warning("Replica %d could not persist request: %s", instance.id, err)
		return
	}
	instance.consumer.StoreState("req."+digest, raw)
}

func (instance *pbftCore) persistDelRequest(digest string) {
	instance.consumer.DelState(digest)
}

func (instance *pbftCore) persistDelAllRequests() {
	reqs, err := instance.consumer.ReadStateSet("req.")
	if err == nil {
		for k := range reqs {
			instance.consumer.DelState(k)
		}
	}
}

func (instance *pbftCore) restoreState() {
	updateSeqView := func(set []*ViewChange_PQ) {
		for _, e := range set {
			if instance.view < e.View {
				instance.view = e.View
			}
			if instance.seqNo < e.SequenceNumber {
				instance.seqNo = e.SequenceNumber
			}
		}
	}

	set := instance.restorePQSet("pset")
	for _, e := range set {
		instance.pset[e.SequenceNumber] = e
	}
	updateSeqView(set)

	set = instance.restorePQSet("qset")
	for _, e := range set {
		instance.qset[qidx{e.Digest, e.SequenceNumber}] = e
	}
	updateSeqView(set)

	reqs, err := instance.consumer.ReadStateSet("req.")
	if err == nil {
		for k, v := range reqs {
			req := &Request{}
			err = proto.Unmarshal(v, req)
			if err != nil {
				logger.Warning("Replica %d could not restore request %s", instance.id, k)
			} else {
				instance.reqStore[hashReq(req)] = req
			}
		}
	} else {
		logger.Warning("Replica %d could not restore reqStore: %s", instance.id, err)
	}

	ok := false
	if instance.lastExec, ok = instance.consumer.getLastSeqNo(); !ok {
		logger.Warning("Replica %d could not restore lastExec", instance.id)
		instance.lastExec = 0
	}

	logger.Info("Replica %d restored state: view: %d, seqNo: %d, lastExec: %d, pset: %d, qset: %d, reqs: %d",
		instance.id, instance.view, instance.seqNo, instance.lastExec, len(instance.pset), len(instance.qset), len(instance.reqStore))
}