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

import "time"

func (s *SBFT) Request(req []byte) {
	s.broadcast(&Msg{&Msg_Request{&Request{req}}})
}

func (s *SBFT) handleRequest(req *Request, src uint64) {
	if s.isPrimary() {
		s.batch = append(s.batch, req)
		if s.batchSize() >= s.config.BatchSizeBytes {
			s.batchReady()
		} else {
			s.startBatchTimer()
		}
	}
}

////////////////////////////////////////////////

func (s *SBFT) startBatchTimer() {
	if s.batchTimer == nil {
		s.batchTimer = s.sys.Timer(time.Duration(s.config.BatchDurationNsec), s.batchReady)
	}
}

func (s *SBFT) batchSize() uint64 {
	size := uint64(0)
	for _, req := range s.batch {
		size += uint64(len(req.Payload))
	}
	return size
}

func (s *SBFT) batchReady() {
	if s.batchTimer != nil {
		s.batchTimer.Cancel()
		s.batchTimer = nil
	}

	batch := s.batch
	s.batch = nil
	s.sendPreprepare(batch)
}