// Copyright 2020 Ohio Supercomputer Center
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log/slog"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	podStart, _  = time.Parse("01/02/2006 15:04:05", "01/01/2020 13:00:00")
	podStartTime = metav1.NewTime(podStart)
)

func clientset() kubernetes.Interface {
	clientset := fake.NewSimpleClientset(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "non-job",
		},
	}, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user-user1",
			Labels: map[string]string{
				"app.kubernetes.io/name": "open-ondemand",
			},
		},
	}, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user-user2",
			Labels: map[string]string{
				"app.kubernetes.io/name": "foo",
			},
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-job-pod",
			Namespace: "non-job",
			Annotations: map[string]string{
				"pod.kubernetes.io/lifetime": "1h",
			},
			Labels: map[string]string{},
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ondemand-job1",
			Namespace: "user-user1",
			Annotations: map[string]string{
				"pod.kubernetes.io/lifetime": "1h",
			},
			Labels: map[string]string{
				"job":                          "1",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
			CreationTimestamp: podStartTime,
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ondemand-user1-job5",
			Namespace: "user-user1",
			Annotations: map[string]string{
				"pod.kubernetes.io/lifetime": "3h",
			},
			Labels: map[string]string{
				"job":                          "5",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
			CreationTimestamp: podStartTime,
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ondemand-job2",
			Namespace: "user-user2",
			Annotations: map[string]string{
				"pod.kubernetes.io/lifetime": "30m",
			},
			Labels: map[string]string{
				"job":                          "2",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
			CreationTimestamp: podStartTime,
		},
	}, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ondemand-job3",
			Namespace: "user-user3",
			Annotations: map[string]string{
				"pod.kubernetes.io/lifetime": "30m",
			},
			Labels: map[string]string{
				"job":                          "3",
				"app.kubernetes.io/managed-by": "test",
			},
			CreationTimestamp: podStartTime,
		},
	}, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-job1",
			Namespace: "user-user1",
			Labels: map[string]string{
				"job": "1",
			},
		},
	}, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-user1-job5",
			Namespace: "user-user1",
			Labels: map[string]string{
				"job":                          "5",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
		},
	}, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-job2",
			Namespace: "user-user2",
			Labels: map[string]string{
				"job": "2",
			},
		},
	}, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-job4",
			Namespace: "user-user2",
			Labels: map[string]string{
				"job":                          "4",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
		},
	}, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-job1",
			Namespace: "user-user1",
			Labels: map[string]string{
				"job": "1",
			},
		},
	}, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-job2",
			Namespace: "user-user2",
			Labels: map[string]string{
				"job": "2",
			},
		},
	}, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-job4",
			Namespace: "user-user2",
			Labels: map[string]string{
				"job":                          "4",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
		},
	}, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-job1",
			Namespace: "user-user1",
			Labels: map[string]string{
				"job": "1",
			},
		},
	}, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-job2",
			Namespace: "user-user2",
			Labels: map[string]string{
				"job": "2",
			},
		},
	}, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-job4",
			Namespace: "user-user2",
			Labels: map[string]string{
				"job":                          "4",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
		},
	})
	return clientset
}

func TestGetNamespaces(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	clientset := clientset()
	namespaces, err := getNamespaces(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(namespaces) != 1 {
		t.Errorf("Unexpected number of namespaces: %d", len(namespaces))
	}
	if namespaces[0] != metav1.NamespaceAll {
		t.Errorf("Unexpected namespace, got: %v", namespaces[0])
	}
}

func TestGetNamespacesByLabel(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--namespace-labels=app.kubernetes.io/name=open-ondemand"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	clientset := clientset()
	namespaces, err := getNamespaces(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(namespaces) != 1 {
		t.Errorf("Unexpected number of namespaces: %d", len(namespaces))
	}
	if namespaces[0] != "user-user1" {
		t.Errorf("Unexpected namespace, got: %v", namespaces[0])
	}
}

func TestGetJobs(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--object-labels=app.kubernetes.io/managed-by=open-ondemand"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 15:00:00")
		return t
	}

	clientset := clientset()
	namespaces, err := getNamespaces(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	jobs, jobIDs, err := getJobs(clientset, namespaces, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
		return
	}
	if val := jobs[0].jobID; val != "1" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
	if val := jobs[1].jobID; val != "2" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
	if len(jobIDs) != 3 {
		t.Errorf("Unexpected number jobIDs, got %d", len(jobIDs))
		return
	}
	expectedJobIDs := []string{"1", "2", "5"}
	sort.Strings(jobIDs)
	sort.Strings(expectedJobIDs)
	if !reflect.DeepEqual(jobIDs, expectedJobIDs) {
		t.Errorf("Unexpected value for jobIDs\nExpected %v\nGot %v\n", expectedJobIDs, jobIDs)
	}
}

func TestGetJobsCase1(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--object-labels=app.kubernetes.io/managed-by=open-ondemand"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 13:45:00")
		return t
	}

	clientset := clientset()
	namespaces, err := getNamespaces(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	jobs, _, err := getJobs(clientset, namespaces, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Expected 1 jobs, got %d", len(jobs))
		return
	}
	if val := jobs[0].jobID; val != "2" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
}

func TestGetJobsNoPodLabels(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 15:00:00")
		return t
	}

	clientset := clientset()
	namespaces, err := getNamespaces(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	jobs, _, err := getJobs(clientset, namespaces, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
		return
	}
	if val := jobs[0].jobID; val != "1" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
	if val := jobs[1].jobID; val != "2" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
	if val := jobs[2].jobID; val != "3" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
}

func TestGetJobsNamespaceLabels(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--namespace-labels=app.kubernetes.io/name=open-ondemand"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 15:00:00")
		return t
	}

	clientset := clientset()
	namespaces, err := getNamespaces(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	jobs, _, err := getJobs(clientset, namespaces, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Expected 1 jobs, got %d", len(jobs))
		return
	}
	if val := jobs[0].jobID; val != "1" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
}

func TestGetJobsNoJobLabel(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--job-label=none"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 15:00:00")
		return t
	}

	clientset := clientset()
	namespaces, err := getNamespaces(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	jobs, jobIDs, err := getJobs(clientset, namespaces, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(jobs) != 4 {
		t.Errorf("Expected 4 jobs, got %d", len(jobs))
		return
	}
	if val := jobs[0].jobID; val != "none" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
	if val := jobs[1].jobID; val != "none" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
	if val := jobs[2].jobID; val != "none" {
		t.Errorf("Unexpected jobID, got: %v", val)
	}
	if len(jobIDs) != 1 {
		t.Errorf("Unexpected number jobIDs, got %d", len(jobIDs))
		return
	}
	expectedJobIDs := []string{"none"}
	if !reflect.DeepEqual(jobIDs, expectedJobIDs) {
		t.Errorf("Unexpected value for jobIDs\nExpected %v\nGot %v\n", expectedJobIDs, jobIDs)
	}
}

func TestRunOnDemand(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--object-labels=app.kubernetes.io/managed-by=open-ondemand"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 15:00:00")
		return t
	}

	resetCounters()
	clientset := clientset()
	err := run(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	pods, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting pods: %v", err)
	}
	if len(pods.Items) != 3 {
		t.Errorf("Unexpected number of pods, got: %d", len(pods.Items))
	}
	services, err := clientset.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting services: %v", err)
	}
	if len(services.Items) != 1 {
		t.Errorf("Unexpected number of services, got: %d", len(services.Items))
	}
	configmaps, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting configmaps: %v", err)
	}
	if len(configmaps.Items) != 0 {
		t.Errorf("Unexpected number of services, got: %d", len(configmaps.Items))
	}
	secrets, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting secrets: %v", err)
	}
	if len(secrets.Items) != 0 {
		t.Errorf("Unexpected number of secrets, got: %d", len(secrets.Items))
	}

	expected := `
	# HELP job_pod_reaper_error Indicates an error was encountered
	# TYPE job_pod_reaper_error gauge
	job_pod_reaper_error 0
	# HELP job_pod_reaper_errors_total Total number of errors
	# TYPE job_pod_reaper_errors_total counter
	job_pod_reaper_errors_total 0
	# HELP job_pod_reaper_reaped_total Total number of object types reaped
	# TYPE job_pod_reaper_reaped_total counter
	job_pod_reaper_reaped_total{type="configmap"} 3
	job_pod_reaper_reaped_total{type="pod"} 2
	job_pod_reaper_reaped_total{type="secret"} 3
	job_pod_reaper_reaped_total{type="service"} 3
	`

	if err := testutil.GatherAndCompare(metricGathers(), strings.NewReader(expected),
		"job_pod_reaper_reaped_total", "job_pod_reaper_error", "job_pod_reaper_errors_total"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestRun(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 15:00:00")
		return t
	}

	resetCounters()
	clientset := clientset()
	err := run(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	pods, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting pods: %v", err)
	}
	if len(pods.Items) != 2 {
		t.Errorf("Unexpected number of pods, got: %d", len(pods.Items))
	}
	services, err := clientset.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting services: %v", err)
	}
	if len(services.Items) != 1 {
		t.Errorf("Unexpected number of services, got: %d", len(services.Items))
	}
	configmaps, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting configmaps: %v", err)
	}
	if len(configmaps.Items) != 0 {
		t.Errorf("Unexpected number of services, got: %d", len(configmaps.Items))
	}
	secrets, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting secrets: %v", err)
	}
	if len(secrets.Items) != 0 {
		t.Errorf("Unexpected number of secrets, got: %d", len(secrets.Items))
	}

	expected := `
	# HELP job_pod_reaper_error Indicates an error was encountered
	# TYPE job_pod_reaper_error gauge
	job_pod_reaper_error 0
	# HELP job_pod_reaper_errors_total Total number of errors
	# TYPE job_pod_reaper_errors_total counter
	job_pod_reaper_errors_total 0
	# HELP job_pod_reaper_reaped_total Total number of object types reaped
	# TYPE job_pod_reaper_reaped_total counter
	job_pod_reaper_reaped_total{type="configmap"} 3
	job_pod_reaper_reaped_total{type="pod"} 3
	job_pod_reaper_reaped_total{type="secret"} 3
	job_pod_reaper_reaped_total{type="service"} 3
	`

	if err := testutil.GatherAndCompare(metricGathers(), strings.NewReader(expected),
		"job_pod_reaper_reaped_total", "job_pod_reaper_error", "job_pod_reaper_errors_total"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestRunNoJobLabel(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--job-label=none"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	timeNow = func() time.Time {
		t, _ := time.Parse("01/02/2006 15:04:05", "01/01/2020 15:00:00")
		return t
	}

	resetCounters()
	clientset := clientset()
	err := run(clientset, logger)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	pods, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting pods: %v", err)
	}
	if len(pods.Items) != 1 {
		t.Errorf("Unexpected number of pods, got: %d", len(pods.Items))
	}
	services, err := clientset.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting services: %v", err)
	}
	if len(services.Items) != 4 {
		t.Errorf("Unexpected number of services, got: %d", len(services.Items))
	}
	configmaps, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting configmaps: %v", err)
	}
	if len(configmaps.Items) != 3 {
		t.Errorf("Unexpected number of services, got: %d", len(configmaps.Items))
	}
	secrets, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Unexpected error getting secrets: %v", err)
	}
	if len(secrets.Items) != 3 {
		t.Errorf("Unexpected number of secrets, got: %d", len(secrets.Items))
	}

	expected := `
	# HELP job_pod_reaper_error Indicates an error was encountered
	# TYPE job_pod_reaper_error gauge
	job_pod_reaper_error 0
	# HELP job_pod_reaper_errors_total Total number of errors
	# TYPE job_pod_reaper_errors_total counter
	job_pod_reaper_errors_total 0
	# HELP job_pod_reaper_reaped_total Total number of object types reaped
	# TYPE job_pod_reaper_reaped_total counter
	job_pod_reaper_reaped_total{type="configmap"} 0
	job_pod_reaper_reaped_total{type="pod"} 4
	job_pod_reaper_reaped_total{type="secret"} 0
	job_pod_reaper_reaped_total{type="service"} 0
	`

	if err := testutil.GatherAndCompare(metricGathers(), strings.NewReader(expected),
		"job_pod_reaper_reaped_total", "job_pod_reaper_error", "job_pod_reaper_errors_total"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestOrphanedFutureResourcesIgnored(t *testing.T) {

	futureTime, _ := time.Parse("01/02/2006 15:04:05", "01/01/2999 23:59:59")
	clientset := fake.NewSimpleClientset(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "future",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
		},
	}, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service",
			Namespace: "future",
			Labels: map[string]string{
				"job":                          "1",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
			CreationTimestamp: metav1.NewTime(futureTime),
		},
	}, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap",
			Namespace: "future",
			Labels: map[string]string{
				"job":                          "1",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
			CreationTimestamp: metav1.NewTime(futureTime),
		},
	}, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "future",
			Labels: map[string]string{
				"job":                          "1",
				"app.kubernetes.io/managed-by": "open-ondemand",
			},
			CreationTimestamp: metav1.NewTime(futureTime),
		},
	})

	if _, err := kingpin.CommandLine.Parse([]string{"--namespace-labels=app.kubernetes.io/name=open-ondemand"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	orphanedObjects, err := getOrphanedJobObjects(clientset, []podJob{}, []string{}, []string{"future"}, logger)

	if err != nil {
		t.Errorf("Not supposed to have error during orphaned job calculation: %v", err)
	}

	if len(orphanedObjects) != 0 {
		t.Errorf("objects from the future cannot be orphaned: %v", orphanedObjects)
	}
}

func resetCounters() {
	metricReapedTotal.Reset()
	metricReapedTotal.WithLabelValues("pod")
	metricReapedTotal.WithLabelValues("service")
	metricReapedTotal.WithLabelValues("configmap")
	metricReapedTotal.WithLabelValues("secret")
}
