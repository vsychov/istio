// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ready

import (
	"testing"

	. "github.com/onsi/gomega"

	"istio.io/istio/pilot/cmd/pilot-agent/status/testserver"
)

var (
	liveServerStats = "cluster_manager.cds.update_success: 1\nlistener_manager.lds.update_success: 1\nserver.state: 0\nlistener_manager.workers_started: 1"
	onlyServerStats = "server.state: 0"
	initServerStats = "cluster_manager.cds.update_success: 1\nlistener_manager.lds.update_success: 1\nserver.state: 2"
	noServerStats   = ""
)

func TestEnvoyStatsCompleteAndSuccessful(t *testing.T) {
	g := NewWithT(t)

	server := testserver.CreateAndStartServer(liveServerStats, "127.0.0.1:1234")
	defer server.Close()
	probe := Probe{AdminPort: 1234}

	err := probe.Check()

	g.Expect(err).NotTo(HaveOccurred())
}

func TestEnvoyStats(t *testing.T) {
	prefix := "config not received from Pilot (is Pilot running?): "
	cases := []struct {
		name   string
		stats  string
		result string
	}{
		{
			"only lds",
			"listener_manager.lds.update_success: 1",
			prefix + "cds updates: 0 successful, 0 rejected; lds updates: 1 successful, 0 rejected",
		},
		{
			"only cds",
			"cluster_manager.cds.update_success: 1",
			prefix + "cds updates: 1 successful, 0 rejected; lds updates: 0 successful, 0 rejected",
		},
		{
			"reject CDS",
			`cluster_manager.cds.update_rejected: 1
listener_manager.lds.update_success: 1`,
			prefix + "cds updates: 0 successful, 1 rejected; lds updates: 1 successful, 0 rejected",
		},
		{
			"workers not started",
			`
cluster_manager.cds.update_success: 1
listener_manager.lds.update_success: 1
listener_manager.workers_started: 0
server.state: 0`,
			"workers have not yet started",
		},
		{
			"full",
			`
cluster_manager.cds.update_success: 1
listener_manager.lds.update_success: 1
listener_manager.workers_started: 1
server.state: 0`,
			"",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.CreateAndStartServer(tt.stats, "127.0.0.1:1234")
			defer server.Close()
			probe := Probe{AdminPort: 1234}

			err := probe.Check()

			// Expect no error
			if tt.result == "" {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
				return
			}
			// Expect error
			if err.Error() != tt.result {
				t.Fatalf("Expected: \n'%v', got: \n'%v'", tt.result, err.Error())
			}
		})
	}
}

func TestEnvoyInitializing(t *testing.T) {
	g := NewWithT(t)

	server := testserver.CreateAndStartServer(initServerStats, "127.0.0.1:1234")
	defer server.Close()
	probe := Probe{AdminPort: 1234}

	err := probe.Check()

	g.Expect(err).To(HaveOccurred())
}

func TestEnvoyNoClusterManagerStats(t *testing.T) {
	g := NewWithT(t)

	server := testserver.CreateAndStartServer(onlyServerStats, "127.0.0.1:1234")
	defer server.Close()
	probe := Probe{AdminPort: 1234}

	err := probe.Check()

	g.Expect(err).To(HaveOccurred())
}

func TestEnvoyNoServerStats(t *testing.T) {
	g := NewWithT(t)

	server := testserver.CreateAndStartServer(noServerStats, "127.0.0.1:1234")
	defer server.Close()
	probe := Probe{AdminPort: 1234}

	err := probe.Check()

	g.Expect(err).To(HaveOccurred())
}

func TestEnvoyReadinessCache(t *testing.T) {
	g := NewWithT(t)

	server := testserver.CreateAndStartServer(noServerStats, "127.0.0.1:1234")
	probe := Probe{AdminPort: 1234}
	err := probe.Check()
	g.Expect(err).To(HaveOccurred())
	g.Expect(probe.atleastOnceReady).Should(BeFalse())
	err = probe.Check()
	g.Expect(err).To(HaveOccurred())
	g.Expect(probe.atleastOnceReady).Should(BeFalse())
	server.Close()

	server = testserver.CreateAndStartServer(liveServerStats, "127.0.0.1:1234")
	err = probe.Check()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(probe.atleastOnceReady).Should(BeTrue())
	server.Close()

	server = testserver.CreateAndStartServer(noServerStats, "127.0.0.1:1234")
	err = probe.Check()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(probe.atleastOnceReady).Should(BeTrue())
	server.Close()

	err = probe.Check()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(probe.atleastOnceReady).Should(BeTrue())
}
