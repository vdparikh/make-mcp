package hosted

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vdparikh/make-mcp/backend/internal/database"
	"github.com/vdparikh/make-mcp/backend/internal/generator"
	"github.com/vdparikh/make-mcp/backend/internal/hostedruntime"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// K8sManager runs hosted MCP servers as Pods + Services in Kubernetes (no Docker socket).
type K8sManager struct {
	clientset         kubernetes.Interface
	namespace         string
	db                *database.DB
	gen               *generator.Generator
	containerBindHost string
	// nodeGeneratedRoot is the host path where generated-servers/<user>/<server>/<version> is stored
	// (must match the API pod's hostPath mount, e.g. /var/lib/make-mcp/generated-servers).
	nodeGeneratedRoot string
}

var _ Runtime = (*K8sManager)(nil)

// NewK8sManager builds a hosted runtime backed by Pods. nodeGeneratedRoot must be the absolute host path
// backing generated code (same as the directory mounted at .../generated-servers in the API pod).
func NewK8sManager(db *database.DB, namespace, nodeGeneratedRoot, containerBindHost, generatedServerPublicHostIP string) (*K8sManager, error) {
	if strings.TrimSpace(namespace) == "" {
		return nil, fmt.Errorf("kubernetes namespace is required")
	}
	if strings.TrimSpace(nodeGeneratedRoot) == "" {
		return nil, fmt.Errorf("hosted.kubernetes.node_generated_root is required")
	}
	if strings.TrimSpace(containerBindHost) == "" {
		return nil, fmt.Errorf("containerBindHost is required")
	}
	if strings.TrimSpace(generatedServerPublicHostIP) == "" {
		return nil, fmt.Errorf("generatedServerPublicHostIP is required")
	}
	cfg, err := restConfig()
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}
	return &K8sManager{
		clientset:         cs,
		namespace:         strings.TrimSpace(namespace),
		db:                db,
		gen:               generator.NewGeneratorWithPublicHost(generatedServerPublicHostIP),
		containerBindHost: strings.TrimSpace(containerBindHost),
		nodeGeneratedRoot: filepath.Clean(nodeGeneratedRoot),
	}, nil
}

func restConfig() (*rest.Config, error) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return rest.InClusterConfig()
	}
	kube := strings.TrimSpace(os.Getenv("KUBECONFIG"))
	if kube == "" {
		home, _ := os.UserHomeDir()
		kube = filepath.Join(home, ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kube)
}

func hostedResourceName(userID, serverID string) string {
	h := sha256.Sum256([]byte(userID + "\x00" + serverID))
	return "mcp-h-" + hex.EncodeToString(h[:8])
}

func serviceDNSName(name, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace)
}

// serviceSelectorLabels matches hosted Pods without version so the Service stays stable across deploys.
func serviceSelectorLabels(userID, serverID string) map[string]string {
	return map[string]string{
		labelManaged: "true",
		labelUserID:  userID,
		labelServer:  serverID,
		"app":        "make-mcp-hosted",
	}
}

func selectorMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (k *K8sManager) ensureService(ctx context.Context, name string, selector map[string]string, objectLabels map[string]string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k.namespace,
			Labels:    objectLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       3000,
					TargetPort: intstr.FromInt(3000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	_, err := k.clientset.CoreV1().Services(k.namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsAlreadyExists(err) {
		return fmt.Errorf("create service: %w", err)
	}
	existing, gerr := k.clientset.CoreV1().Services(k.namespace).Get(ctx, name, metav1.GetOptions{})
	if gerr != nil {
		return gerr
	}
	if selectorMapsEqual(existing.Spec.Selector, selector) {
		return nil
	}
	_ = k.clientset.CoreV1().Services(k.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	_, err = k.clientset.CoreV1().Services(k.namespace).Create(ctx, svc, metav1.CreateOptions{})
	return err
}

// EnsureContainer implements Runtime.
func (k *K8sManager) EnsureContainer(ctx context.Context, userID string, serverID string, version string, snapshot *models.Server, envVars map[string]string, rt *hostedruntime.Resolved, idleTimeoutMinutes int) (*ContainerConfig, error) {
	if userID == "" || snapshot == nil {
		return nil, fmt.Errorf("userID and snapshot are required")
	}
	if serverID == "" || version == "" {
		return nil, fmt.Errorf("serverID and version are required")
	}
	if rt == nil {
		def, err := hostedruntime.Resolve(hostedruntime.UserConfig{}, hostedruntime.DefaultPlatformLimits())
		if err != nil {
			return nil, err
		}
		rt = def
	}
	requestedIdleTimeout := idleTimeoutMinutes
	if requestedIdleTimeout < 0 {
		requestedIdleTimeout = 0
	}

	if cfg, err := k.reconcileK8s(ctx, userID, serverID, version, envVars, requestedIdleTimeout); err == nil && cfg != nil {
		cfg.LastUsedAt = time.Now()
		_ = k.upsertSession(ctx, userID, serverID, *cfg, "running", "unknown", "", nil)
		return cfg, nil
	} else if err != nil {
		return nil, err
	}

	gen, err := k.gen.Generate(snapshot)
	if err != nil {
		return nil, fmt.Errorf("generate server: %w", err)
	}

	rootDir := filepath.Join("generated-servers", userID, serverID, version)
	if err := writeGeneratedServer(rootDir, gen); err != nil {
		return nil, fmt.Errorf("write generated server: %w", err)
	}
	if err := writeHostedManifest(rootDir, snapshot, version, userID, serverID, envVars, requestedIdleTimeout, k.containerBindHost, rt, "kubernetes"); err != nil {
		return nil, fmt.Errorf("write hosted manifest: %w", err)
	}

	cfg, err := k.startPod(ctx, userID, serverID, version, envVars, requestedIdleTimeout, rt)
	if err != nil {
		return nil, err
	}
	_ = k.upsertSession(ctx, userID, serverID, *cfg, "running", "unknown", "", nil)
	return cfg, nil
}

func (k *K8sManager) reconcileK8s(ctx context.Context, userID, serverID, version string, envVars map[string]string, idleTimeoutMinutes int) (*ContainerConfig, error) {
	name := hostedResourceName(userID, serverID)
	pod, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if pod.Labels[labelVersion] != version {
		_ = k.deletePodAndService(ctx, name)
		return nil, nil
	}
	if idleTimeoutMinutes >= 0 && pod.Labels[labelIdleTimeoutMinutes] != strconv.Itoa(idleTimeoutMinutes) {
		_ = k.deletePodAndService(ctx, name)
		return nil, nil
	}
	if !podEnvMatches(pod, envVars) {
		_ = k.deletePodAndService(ctx, name)
		return nil, nil
	}
	switch pod.Status.Phase {
	case corev1.PodFailed, corev1.PodSucceeded:
		_ = k.deletePodAndService(ctx, name)
		return nil, nil
	case corev1.PodRunning:
		// ok
	default:
		dial := serviceDNSName(name, k.namespace)
		if err := WaitForHostedHTTPReady(ctx, dial, "3000"); err != nil {
			return nil, err
		}
		pod, err = k.clientset.CoreV1().Pods(k.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if pod.Status.Phase != corev1.PodRunning {
			return nil, fmt.Errorf("hosted pod %s/%s is not running (phase=%s)", k.namespace, name, pod.Status.Phase)
		}
	}
	dial := serviceDNSName(name, k.namespace)
	cfg := &ContainerConfig{
		ContainerID:        string(pod.UID),
		HostPort:           "3000",
		DialHost:           dial,
		Version:            version,
		StartedAt:          pod.CreationTimestamp.Time,
		LastUsedAt:         time.Now(),
		IdleTimeoutMinutes: idleTimeoutMinutes,
	}
	return cfg, nil
}

func podEnvMatches(pod *corev1.Pod, expected map[string]string) bool {
	if len(expected) == 0 {
		return true
	}
	envMap := make(map[string]string)
	for _, c := range pod.Spec.Containers {
		for _, e := range c.Env {
			if e.Value != "" {
				envMap[e.Name] = e.Value
			}
		}
		break
	}
	for k, v := range expected {
		if envMap[k] != v {
			return false
		}
	}
	return true
}

func (k *K8sManager) startPod(ctx context.Context, userID, serverID, version string, envVars map[string]string, idleTimeoutMinutes int, rt *hostedruntime.Resolved) (*ContainerConfig, error) {
	name := hostedResourceName(userID, serverID)
	hostDir := filepath.Join(k.nodeGeneratedRoot, userID, serverID, version)

	memQty := resource.NewQuantity(rt.MemoryBytes, resource.BinarySI)
	cpuMillis := int64(rt.NanoCPUs / 1e6)
	if cpuMillis < 1 {
		cpuMillis = 1
	}
	cpuQty := resource.NewMilliQuantity(cpuMillis, resource.DecimalSI)
	reqCpuMillis := cpuMillis / 2
	if reqCpuMillis < 1 {
		reqCpuMillis = 1
	}
	reqCpuQty := resource.NewMilliQuantity(reqCpuMillis, resource.DecimalSI)

	var env []corev1.EnvVar
	for key, val := range envVars {
		if key == "" {
			continue
		}
		env = append(env, corev1.EnvVar{Name: key, Value: val})
	}

	sel := serviceSelectorLabels(userID, serverID)
	labels := map[string]string{
		labelVersion:            version,
		labelIdleTimeoutMinutes: strconv.Itoa(idleTimeoutMinutes),
		labelIsolationTier:      rt.Tier,
	}
	for k, v := range sel {
		labels[k] = v
	}

	cmd := "cd /app && npm install && npm run build && MCP_TRANSPORT=http MCP_HTTP_PORT=3000 node dist/server.js"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k.namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "mcp",
					Image:   hostedRuntimeImage,
					Command: []string{"sh", "-lc", cmd},
					Ports: []corev1.ContainerPort{
						{ContainerPort: 3000, Protocol: corev1.ProtocolTCP},
					},
					WorkingDir: "/app",
					Env:        env,
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory:           *memQty,
							corev1.ResourceCPU:              *cpuQty,
							corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: *resource.NewQuantity(rt.MemoryBytes/2, resource.BinarySI),
							corev1.ResourceCPU:    *reqCpuQty,
						},
					},
					// hostPath is populated by the API pod (typically root); non-root UIDs cannot mkdir node_modules here.
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:                func() *int64 { u := int64(0); return &u }(),
						RunAsNonRoot:             func() *bool { b := false; return &b }(),
						AllowPrivilegeEscalation: func() *bool { b := false; return &b }(),
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "src", MountPath: "/app", ReadOnly: false},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "src",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: hostDir,
							Type: func() *corev1.HostPathType { t := corev1.HostPathDirectory; return &t }(),
						},
					},
				},
			},
		},
	}

	if err := k.ensureService(ctx, name, sel, labels); err != nil {
		return nil, err
	}
	_, err := k.clientset.CoreV1().Pods(k.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			_ = k.clientset.CoreV1().Services(k.namespace).Delete(ctx, name, metav1.DeleteOptions{})
			return nil, fmt.Errorf("create pod: %w", err)
		}
	}

	dial := serviceDNSName(name, k.namespace)
	if err := WaitForHostedHTTPReady(ctx, dial, "3000"); err != nil {
		_ = k.deletePodAndService(ctx, name)
		return nil, err
	}

	p, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &ContainerConfig{
		ContainerID:        string(p.UID),
		HostPort:           "3000",
		DialHost:           dial,
		Version:            version,
		StartedAt:          p.CreationTimestamp.Time,
		LastUsedAt:         time.Now(),
		IdleTimeoutMinutes: idleTimeoutMinutes,
	}, nil
}

func (k *K8sManager) deletePodAndService(ctx context.Context, name string) error {
	_ = k.clientset.CoreV1().Pods(k.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	_ = k.clientset.CoreV1().Services(k.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return nil
}

// GetContainerForServer implements Runtime.
func (k *K8sManager) GetContainerForServer(ctx context.Context, userID, serverID string) (*ContainerConfig, error) {
	name := hostedResourceName(userID, serverID)
	pod, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if pod.Status.Phase != corev1.PodRunning {
		return nil, nil
	}
	idle := parseContainerIdleTimeout(pod.Labels[labelIdleTimeoutMinutes])
	if idle > 0 && k.db != nil {
		if s, err := k.db.GetHostedSession(ctx, userID, serverID); err == nil && s != nil && s.LastUsedAt != nil {
			if time.Since(*s.LastUsedAt) >= time.Duration(idle)*time.Minute {
				_, _ = k.StopSession(ctx, userID, serverID)
				return nil, nil
			}
		}
	}
	dial := serviceDNSName(name, k.namespace)
	return &ContainerConfig{
		ContainerID:        string(pod.UID),
		HostPort:           "3000",
		DialHost:           dial,
		Version:            pod.Labels[labelVersion],
		StartedAt:          pod.CreationTimestamp.Time,
		LastUsedAt:         time.Now(),
		IdleTimeoutMinutes: idle,
	}, nil
}

// ListSessions implements Runtime.
func (k *K8sManager) ListSessions(ctx context.Context, userID string) ([]models.HostedSession, error) {
	if k.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	sessions, err := k.db.ListHostedSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.HostedSession, 0, len(sessions))
	for _, s := range sessions {
		updated := s
		if strings.TrimSpace(s.ContainerID) == "" {
			out = append(out, updated)
			continue
		}
		name := hostedResourceName(s.UserID, s.ServerID)
		pod, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, name, metav1.GetOptions{})
		now := time.Now().UTC()
		if err != nil {
			updated.Status = "stopped"
			updated.Health = "unknown"
			updated.StoppedAt = &now
			updated.LastError = err.Error()
		} else {
			updated.Status = "stopped"
			if pod.Status.Phase == corev1.PodRunning {
				updated.Status = "running"
			}
			updated.StartedAt = &pod.CreationTimestamp.Time
			updated.Health = "unknown"
			if pod.Status.Phase == corev1.PodRunning {
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						updated.Health = "healthy"
						break
					}
				}
			}
			updated.LastError = ""
			if updated.Status == "stopped" {
				updated.StoppedAt = &now
			}
		}
		if _, err := k.db.UpsertHostedSession(ctx, updated); err != nil {
			return nil, err
		}
		out = append(out, updated)
	}
	return out, nil
}

// StopSession implements Runtime.
func (k *K8sManager) StopSession(ctx context.Context, userID, serverID string) (*models.HostedSession, error) {
	if k.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s, err := k.db.GetHostedSession(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	name := hostedResourceName(userID, serverID)
	_ = k.deletePodAndService(ctx, name)
	s.Status = "stopped"
	s.Health = "unknown"
	s.LastError = ""
	s.StoppedAt = &now
	return k.db.UpsertHostedSession(ctx, *s)
}

// RestartSession implements Runtime.
func (k *K8sManager) RestartSession(ctx context.Context, userID, serverID string) (*models.HostedSession, error) {
	if k.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s, err := k.db.GetHostedSession(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	sv, err := k.db.GetServerVersion(ctx, serverID, s.SnapshotVersion)
	if err != nil || sv == nil {
		return nil, fmt.Errorf("load server version %s: %w", s.SnapshotVersion, err)
	}
	var snap models.Server
	if err := json.Unmarshal(sv.Snapshot, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot: %w", err)
	}
	_ = k.deletePodAndService(ctx, hostedResourceName(userID, serverID))
	env := map[string]string{}
	cfg, err := k.EnsureContainer(ctx, userID, serverID, s.SnapshotVersion, &snap, env, nil, -1)
	if err != nil {
		s.Status = "error"
		s.LastError = err.Error()
		updated, upsertErr := k.db.UpsertHostedSession(ctx, *s)
		if upsertErr != nil {
			return nil, fmt.Errorf("ensure hosted pod: %w; upsert session: %v", err, upsertErr)
		}
		return updated, nil
	}
	now := time.Now().UTC()
	s.Status = "running"
	s.Health = "unknown"
	s.LastError = ""
	s.ContainerID = cfg.ContainerID
	s.HostPort = cfg.HostPort
	s.LastEnsuredAt = &now
	s.LastUsedAt = &now
	s.StartedAt = &now
	s.StoppedAt = nil
	return k.db.UpsertHostedSession(ctx, *s)
}

// SessionHealth implements Runtime.
func (k *K8sManager) SessionHealth(ctx context.Context, userID, serverID string) (*models.HostedSession, error) {
	if k.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s, err := k.db.GetHostedSession(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	name := hostedResourceName(userID, serverID)
	pod, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, name, metav1.GetOptions{})
	now := time.Now().UTC()
	if err != nil {
		s.Status = "stopped"
		s.Health = "unknown"
		s.StoppedAt = &now
		s.LastError = err.Error()
		return k.db.UpsertHostedSession(ctx, *s)
	}
	s.LastError = ""
	if pod.Status.Phase == corev1.PodRunning {
		s.Status = "running"
		s.StoppedAt = nil
		s.Health = "unknown"
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				s.Health = "healthy"
				break
			}
		}
	} else {
		s.Status = "stopped"
		s.StoppedAt = &now
		s.Health = "unknown"
	}
	return k.db.UpsertHostedSession(ctx, *s)
}

// upsertSession is duplicated from Manager — keep package-local helper.
func (k *K8sManager) upsertSession(ctx context.Context, userID, serverID string, cfg ContainerConfig, status, health, lastError string, stoppedAt *time.Time) error {
	if k.db == nil {
		return nil
	}
	now := time.Now().UTC()
	s := models.HostedSession{
		UserID:          userID,
		ServerID:        serverID,
		SnapshotVersion: cfg.Version,
		ContainerID:     cfg.ContainerID,
		HostPort:        cfg.HostPort,
		Status:          status,
		Health:          health,
		LastUsedAt:      &cfg.LastUsedAt,
		LastEnsuredAt:   &now,
		StartedAt:       &cfg.StartedAt,
		StoppedAt:       stoppedAt,
		LastError:       lastError,
	}
	_, err := k.db.UpsertHostedSession(ctx, s)
	return err
}
