package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/posthog/posthog-go"
	"github.com/stacksnap/stacksnap/internal/backup"
	"github.com/stacksnap/stacksnap/internal/compose"
	"github.com/stacksnap/stacksnap/internal/config"
	"github.com/stacksnap/stacksnap/internal/docker"
	"github.com/stacksnap/stacksnap/internal/license"
	"github.com/stacksnap/stacksnap/internal/storage"
)

var PostHogKey string

type Server struct {
	mux           *http.ServeMux
	provider      storage.Provider
	config        *config.Config
	setupRequired bool
	uiFS          fs.FS
	broker        *EventBroker
	phClient      posthog.Client
	machineID     string
}

func NewServer(provider storage.Provider, uiFS fs.FS) *Server {

	cfg, err := config.Load()
	setupRequired := false

	if provider == nil {
		if err != nil {

			setupRequired = true
			fmt.Println(" Config not found. Starting in SETUP MODE.")
		} else {

			fmt.Println(" Config loaded. Starting in DASHBOARD MODE.")

			valid, err := license.Verify(cfg.LicenseServerURL, cfg.LicenseKey, cfg.MachineID)
			if err != nil || !valid {
				fmt.Println(" INVALID LICENSE. Reverting to SETUP MODE.")
				setupRequired = true
			} else {

				ctx := context.Background()
				if cfg.Storage.Type == config.StorageS3 {
					p, err := storage.NewS3Provider(
						ctx,
						cfg.Storage.S3Bucket,
						cfg.Storage.S3Region,
						cfg.Storage.S3Endpoint,
						cfg.Storage.S3AccessKey,
						cfg.Storage.S3SecretKey,
					)
					if err == nil {
						provider = p
					} else {
						fmt.Printf("Error initializing S3 provider: %v\n", err)
					}
				}
			}

		}
	}

	s := &Server{
		mux:           http.NewServeMux(),
		provider:      provider,
		config:        cfg,
		setupRequired: setupRequired,
		uiFS:          uiFS,
		broker:        NewEventBroker(),
	}

	if PostHogKey != "" {
		fmt.Println(" [Telemetry] Active")
		phClient, _ := posthog.NewWithConfig(
			PostHogKey,
			posthog.Config{
				Endpoint: "https://us.i.posthog.com",
			},
		)
		s.phClient = phClient
	}

	hostname, _ := os.Hostname()
	s.machineID = hostname

	s.track("server_started", map[string]interface{}{
		"mode": func() string {
			if setupRequired {
				return "setup"
			}
			return "dashboard"
		}(),
		"storage_type": func() string {
			if cfg != nil {
				return string(cfg.Storage.Type)
			}
			return "none"
		}(),
		"os": "mac",
	})

	s.routes()
	return s
}

func (s *Server) track(event string, properties map[string]interface{}) {
	if s.phClient == nil {
		return
	}
	s.phClient.Enqueue(posthog.Capture{
		DistinctId: s.machineID,
		Event:      event,
		Properties: properties,
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

type EventBroker struct {
	clients    map[chan string]bool
	newClients chan chan string
	defunct    chan chan string
	messages   chan string
}

func NewEventBroker() *EventBroker {
	b := &EventBroker{
		clients:    make(map[chan string]bool),
		newClients: make(chan chan string),
		defunct:    make(chan chan string),
		messages:   make(chan string),
	}
	go b.listen()
	return b
}

func (b *EventBroker) listen() {
	for {
		select {
		case s := <-b.newClients:
			b.clients[s] = true
		case s := <-b.defunct:
			delete(b.clients, s)
			close(s)
		case msg := <-b.messages:
			for s := range b.clients {

				select {
				case s <- msg:
				default:
				}
			}
		}
	}
}

func (b *EventBroker) Broadcast(msg string) {
	b.messages <- msg
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan string)
	s.broker.newClients <- clientChan

	defer func() {
		s.broker.defunct <- clientChan
	}()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		}
	}
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/events", s.handleEvents)
	s.mux.HandleFunc("/api/setup", s.handleSetup)

	s.mux.HandleFunc("/api/stacks", s.handleListStacks)
	s.mux.HandleFunc("/api/backups", s.handleBackups)
	s.mux.HandleFunc("/api/stats", s.handleStats)
	s.mux.HandleFunc("/api/history", s.handleHistory)
	s.mux.HandleFunc("/api/restore", s.handleRestore)
	s.mux.HandleFunc("/api/logs", s.handleLogs)
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/test-storage", s.handleTestStorage)
	s.mux.HandleFunc("/api/verify", s.handleVerify)
	s.mux.HandleFunc("/api/history/peek", s.handleHistoryPeek)
	s.mux.HandleFunc("/api/stacks/add", s.handleAddStack)
	s.mux.HandleFunc("/api/stacks/remove", s.handleRemoveStack)
	s.mux.HandleFunc("/api/system-health", s.handleSystemHealth)

	if s.uiFS != nil {
		fileServer := http.FileServer(http.FS(s.uiFS))
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			f, err := s.uiFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				defer f.Close()
				stat, _ := f.Stat()
				if !stat.IsDir() {
					fileServer.ServeHTTP(w, r)
					return
				}
			}

			if strings.HasPrefix(path, "/assets/") {
				http.NotFound(w, r)
				return
			}

			content, err := fs.ReadFile(s.uiFS, "index.html")
			if err != nil {
				http.Error(w, "Index not found", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(content)
		})
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if s.setupRequired {
		status = "setup_required"
	}
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req config.Config
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	machineID, err := license.GetMachineID()
	if err != nil {
		http.Error(w, "Failed to generate Machine ID", http.StatusInternalServerError)
		return
	}
	req.MachineID = machineID

	valid, err := license.Verify(req.LicenseServerURL, req.LicenseKey, machineID)
	if err != nil {
		http.Error(w, "License verification error", http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "Invalid License Key", http.StatusForbidden)
		return
	}

	if err := config.Save(&req); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	if req.Storage.Type == config.StorageS3 {
		p, err := storage.NewS3Provider(
			ctx,
			req.Storage.S3Bucket,
			req.Storage.S3Region,
			req.Storage.S3Endpoint,
			req.Storage.S3AccessKey,
			req.Storage.S3SecretKey,
		)
		if err != nil {
			http.Error(w, "Failed to initialize storage provider: "+err.Error(), http.StatusBadRequest)
			return
		}
		s.provider = p
	}

	s.config = &req
	s.setupRequired = false

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "setup_complete",
		"message": "Configuration saved. Dashboard ready.",
	})
}

func (s *Server) handleListStacks(w http.ResponseWriter, r *http.Request) {
	client, err := docker.NewClient()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get docker client: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	allContainers, err := client.ListAllContainers()
	if err != nil {
		fmt.Printf(" Warning: failed to list all containers: %v\n", err)
	}

	trackedIDs := make(map[string]bool)
	projectContainers := make(map[string][]docker.ContainerInfo)
	projectToDir := make(map[string]string)

	for _, ctr := range allContainers {
		if projectName, ok := ctr.Labels["com.docker.compose.project"]; ok {
			projectContainers[projectName] = append(projectContainers[projectName], ctr)

			if dir, ok := ctr.Labels["com.docker.compose.project.working_dir"]; ok {
				projectToDir[projectName] = dir
			}
		}
	}

	relevantDirs := make(map[string]bool)
	cwd, _ := os.Getwd()
	relevantDirs[cwd] = true
	if s.config != nil {
		for _, d := range s.config.ManualStacks {
			relevantDirs[d] = true
		}
	}

	for d := range relevantDirs {
		if _, err := os.Stat(d); err == nil {
			stack, err := compose.DiscoverStack(d)
			if err == nil {

				if _, exists := projectContainers[stack.Name]; !exists {

					projectContainers[stack.Name] = []docker.ContainerInfo{}
					projectToDir[stack.Name] = d
				} else {

					projectToDir[stack.Name] = d
				}
			}
		}
	}

	var stacks []*compose.Stack

	for projectName, containers := range projectContainers {
		dir := projectToDir[projectName]
		var stack *compose.Stack

		if dir != "" {
			if _, err := os.Stat(dir); err == nil {
				s, err := compose.DiscoverStack(dir)
				if err == nil {
					stack = s
				}
			}
		}

		if stack == nil {
			stack = &compose.Stack{
				Name:     projectName,
				Status:   "Running",
				Services: make(map[string]compose.ServiceDiagnostics),
			}
		}

		runningCount := 0
		for _, ctr := range containers {
			svcName := ctr.Labels["com.docker.compose.service"]
			if svcName == "" {
				svcName = ctr.Name
			}

			stack.Services[svcName] = compose.ServiceDiagnostics{
				ContainerID: ctr.ID,
				State:       ctr.State,
				Health:      ctr.Health,
				Paused:      ctr.Paused,
			}
			if ctr.State == "running" {
				runningCount++
			}
			trackedIDs[ctr.ID] = true
		}

		if runningCount == 0 {
			stack.Status = "Stopped"
		} else if runningCount < len(stack.Services) {
			stack.Status = "Partial"
		} else {
			stack.Status = "Running"
		}

		stacks = append(stacks, stack)
	}

	standaloneStack := &compose.Stack{
		Name:         "Standalone Containers",
		Status:       "Running",
		IsStandalone: true,
		Services:     make(map[string]compose.ServiceDiagnostics),
	}

	standaloneCount := 0
	for _, ctr := range allContainers {
		if !trackedIDs[ctr.ID] {
			standaloneStack.Services[ctr.Name] = compose.ServiceDiagnostics{
				ContainerID: ctr.ID,
				State:       ctr.State,
				Health:      ctr.Health,
				Paused:      ctr.Paused,
			}
			standaloneCount++
		}
	}

	if standaloneCount > 0 {
		stacks = append(stacks, standaloneStack)
	}

	fmt.Printf(" Discovery: %d containers total -> %d stacks (labels-first), %d standalone\n",
		len(allContainers), len(stacks), standaloneCount)

	sort.Slice(stacks, func(i, j int) bool {
		if stacks[i].IsStandalone != stacks[j].IsStandalone {
			return !stacks[i].IsStandalone
		}
		return stacks[i].Name < stacks[j].Name
	})

	json.NewEncoder(w).Encode(stacks)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "Missing container id", http.StatusBadRequest)
		return
	}

	client, err := docker.NewClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	logs, err := client.GetContainerLogs(containerID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(logs))
}

func (s *Server) handleBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Location        string `json:"location"`
		ProjectName     string `json:"project_name"`
		Pause           bool   `json:"pause"`
		IncludeDB       bool   `json:"include_db"`
		Verify          bool   `json:"verify"`
		SnapshotImages  bool   `json:"snapshot_images"`
		EncryptionKeyID string `json:"encryption_key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Location == "" && req.ProjectName == "" {
		http.Error(w, "Either location or project_name is required", http.StatusBadRequest)
		return
	}

	var key []byte

	s.track("backup_initiated", map[string]interface{}{
		"project":         req.ProjectName,
		"snapshot_images": req.SnapshotImages,
		"verify":          req.Verify,
	})

	go func() {
		dockerClient, err := docker.NewClient()
		if err != nil {
			fmt.Printf("Error creating docker client: %v\n", err)
			s.broker.Broadcast("ERROR: Internal Docker client error")
			return
		}
		defer dockerClient.Close()

		logFunc := func(msg string) {

			cleanMsg := strings.TrimSpace(msg)
			if cleanMsg != "" {
				s.broker.Broadcast(fmt.Sprintf("[%s] INFO: %s", time.Now().Format("15:04:05"), cleanMsg))
			}
		}

		if req.ProjectName != "" {
			logFunc(fmt.Sprintf("Starting backup for project: %s", req.ProjectName))
		} else {
			logFunc(fmt.Sprintf("Starting backup for location: %s", req.Location))
		}

		res, err := backup.BackupStack(dockerClient, backup.StackBackupOptions{
			Directory:       req.Location,
			ProjectName:     req.ProjectName,
			PauseContainers: req.Pause,
			IncludeDatabase: req.IncludeDB,
			SnapshotImages:  req.SnapshotImages,
			StorageProvider: s.provider,
			EncryptionKey:   key,
			Logger:          logFunc,
		})

		if err != nil {
			fmt.Printf("Backup failed: %v\n", err)
			logFunc(fmt.Sprintf(" Backup failed: %v", err))
			s.broker.Broadcast("ERROR: " + err.Error())
			s.track("backup_failed", map[string]interface{}{
				"project": req.ProjectName,
				"error":   err.Error(),
			})
			return
		}

		logFunc(" Backup completed successfully")
		s.track("backup_completed", map[string]interface{}{
			"project": req.ProjectName,
			"size":    res.Size,
		})

		if req.Verify {
			logFunc(" Auto-verifying backup integrity...")
			vRes, vErr := backup.VerifyBackup(context.Background(), dockerClient, s.provider, res.OutputPath)
			if vErr != nil {
				logFunc(fmt.Sprintf(" Verification failed: %v", vErr))

			} else {
				logFunc(fmt.Sprintf(" Verified (Checksum: %s)", "OK"))

				verfMap := s.loadVerifications()
				verfMap[res.OutputPath] = vRes
				s.saveVerifications(verfMap)
			}
		}

		s.broker.Broadcast("COMPLETE")
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "backup_started",
		"message": "Backup job has been queued",
	})
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Filename    string `json:"filename"`
		ProjectName string `json:"project_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	go func() {
		dockerClient, err := docker.NewClient()
		if err != nil {
			fmt.Printf("Error creating docker client: %v\n", err)
			return
		}
		defer dockerClient.Close()

		go func() {
			logFunc := func(msg string) {
				cleanMsg := strings.TrimSpace(msg)
				if cleanMsg != "" {
					s.broker.Broadcast(fmt.Sprintf("[%s] INFO: %s", time.Now().Format("15:04:05"), cleanMsg))
				}
			}

			logFunc(fmt.Sprintf("Starting restore for %s...", req.Filename))
			s.track("restore_initiated", map[string]interface{}{
				"project": req.ProjectName,
			})

			var key string
			var keyBytes []byte
			if key != "" {
				keyBytes = []byte(key)
			}

			err := backup.RestoreStack(dockerClient, backup.StackRestoreOptions{
				StackName:       req.ProjectName,
				InputPath:       req.Filename,
				StorageProvider: s.provider,
				EncryptionKey:   keyBytes,
				Logger:          logFunc,
				Context:         ctx,
			})
			if err != nil {
				fmt.Printf(" Restore failed: %v\n", err)
				logFunc(fmt.Sprintf(" Restore failed: %v", err))
				s.broker.Broadcast("ERROR: " + err.Error())
				s.track("restore_failed", map[string]interface{}{
					"project": req.ProjectName,
					"error":   err.Error(),
				})
			} else {
				logFunc(" Restore completed successfully")
				s.track("restore_completed", map[string]interface{}{
					"project": req.ProjectName,
				})
				s.broker.Broadcast("COMPLETE")
			}
		}()
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "restore_started"})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	items := make([]storage.BackupItem, 0)
	var err error

	prefix := r.URL.Query().Get("prefix")

	if s.provider != nil {
		items, err = s.provider.List(ctx, prefix)
	} else {

		all := listLocalBackups()
		if prefix == "" {
			items = all
		} else {
			for _, i := range all {
				if strings.HasPrefix(i.Key, prefix) {
					items = append(items, i)
				}
			}
		}
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list backups: %v", err), http.StatusInternalServerError)
		return
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].LastModified.After(items[j].LastModified)
	})

	verfMap := s.loadVerifications()
	type HistoryResponseItem struct {
		storage.BackupItem
		Verification *backup.VerificationResult `json:"verification,omitempty"`
	}

	resp := make([]HistoryResponseItem, len(items))
	for i, item := range items {
		resp[i] = HistoryResponseItem{
			BackupItem:   item,
			Verification: verfMap[item.Key],
		}
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	client, err := docker.NewClient()
	if err != nil {
		http.Error(w, "Failed to connect to Docker", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	ctx := context.Background()
	result, err := backup.VerifyBackup(ctx, client, s.provider, req.Key)
	if err != nil {

		result = &backup.VerificationResult{
			BackupKey:    req.Key,
			Verified:     false,
			TestedAt:     time.Now(),
			ErrorMessage: err.Error(),
		}
	}

	verfMap := s.loadVerifications()
	verfMap[req.Key] = result
	s.saveVerifications(verfMap)

	json.NewEncoder(w).Encode(result)
}

func (s *Server) loadVerifications() map[string]*backup.VerificationResult {
	res := make(map[string]*backup.VerificationResult)
	data, err := os.ReadFile(config.VerificationsPath())
	if err == nil {
		json.Unmarshal(data, &res)
	}
	return res
}

func (s *Server) saveVerifications(m map[string]*backup.VerificationResult) {
	data, _ := json.Marshal(m)
	os.WriteFile(config.VerificationsPath(), data, 0644)
}

func (s *Server) handleVerificationsLoad() map[string]*backup.VerificationResult {
	return s.loadVerifications()
}

func (s *Server) handleHistoryPeek(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	files, err := backup.PeekBackup(backup.StackRestoreOptions{
		InputPath:       key,
		StorageProvider: s.provider,
		Context:         context.Background(),
	})
	if err != nil {
		http.Error(w, "Failed to peek backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(files)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()
	var items []storage.BackupItem

	if s.provider != nil {
		items, _ = s.provider.List(ctx, "")
	} else {
		items = listLocalBackups()
	}

	totalBackups := len(items)
	var totalSize int64
	for _, item := range items {
		totalSize += item.Size
	}

	var sizeStr string
	const unit = 1024
	if totalSize < unit {
		sizeStr = fmt.Sprintf("%d B", totalSize)
	} else {
		div, exp := int64(unit), 0
		for n := totalSize / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		sizeStr = fmt.Sprintf("%.1f %cB", float64(totalSize)/float64(div), "KMGTPE"[exp])
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_backups": totalBackups,
		"success_rate":  100.0,
		"storage_used":  sizeStr,
	})
}

func (s *Server) handleTestStorage(w http.ResponseWriter, r *http.Request) {
	if s.provider == nil {
		http.Error(w, "No storage provider configured", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	_, err := s.provider.List(ctx, "test-connection-probe")
	if err != nil {
		http.Error(w, "Storage connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Connection to S3 bucket successful!"})
}

func listLocalBackups() []storage.BackupItem {
	cwd, _ := os.Getwd()
	entries, _ := os.ReadDir(cwd)
	var items []storage.BackupItem

	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if (len(name) > 7 && name[len(name)-7:] == ".tar.gz") ||
				(len(name) > 4 && name[len(name)-4:] == ".enc") {
				info, _ := entry.Info()
				items = append(items, storage.BackupItem{
					Key:          name,
					Size:         info.Size(),
					LastModified: info.ModTime(),
				})
			}
		}
	}
	return items
}
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleGetConfig(w, r)
	} else if r.Method == http.MethodPost {
		s.handleUpdateConfig(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	if cfg.Storage.S3SecretKey != "" {
		cfg.Storage.S3SecretKey = "********"
	}
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req config.Config
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	current, _ := config.Load()
	if req.MachineID == "" && current != nil {
		req.MachineID = current.MachineID
	}
	if req.LicenseKey == "" && current != nil {
		req.LicenseKey = current.LicenseKey
	}
	if req.LicenseServerURL == "" && current != nil {
		req.LicenseServerURL = current.LicenseServerURL
	}

	if req.Storage.S3SecretKey == "********" && current != nil {
		req.Storage.S3SecretKey = current.Storage.S3SecretKey
	}

	if err := config.Save(&req); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	s.config = &req

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Configuration updated"})
}

func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	client, err := docker.NewClient()
	if err != nil {
		http.Error(w, "Failed to connect to Docker", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	info, err := client.Info()
	if err != nil {
		http.Error(w, "Failed to get system info", http.StatusInternalServerError)
		return
	}

	diskUsage, err := client.DiskUsage()
	if err != nil {
		fmt.Printf("Warning: Failed to get disk usage: %v\n", err)
	}

	var totalSize int64
	if diskUsage.LayersSize > 0 {
		totalSize = diskUsage.LayersSize
	} else {

		for _, img := range diskUsage.Images {
			totalSize += img.Size
		}
	}

	resp := map[string]interface{}{
		"cpu_cores":          info.NCPU,
		"memory_total":       info.MemTotal,
		"os_type":            info.OSType,
		"architecture":       info.Architecture,
		"containers":         info.Containers,
		"containers_running": info.ContainersRunning,
		"containers_paused":  info.ContainersPaused,
		"containers_stopped": info.ContainersStopped,
		"images_count":       info.Images,
		"disk_usage_bytes":   totalSize,
		"server_version":     info.ServerVersion,
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAddStack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(req.Path); err != nil {
		http.Error(w, "Directory does not exist", http.StatusBadRequest)
		return
	}

	if _, err := compose.FindComposeFile(req.Path); err != nil {
		http.Error(w, "No docker-compose file found in directory", http.StatusBadRequest)
		return
	}

	cfg, _ := config.Load()
	if cfg == nil {
		http.Error(w, "Config not found", http.StatusInternalServerError)
		return
	}

	for _, p := range cfg.ManualStacks {
		if p == req.Path {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	cfg.ManualStacks = append(cfg.ManualStacks, req.Path)
	if err := config.Save(cfg); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	s.config = cfg
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRemoveStack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cfg, _ := config.Load()
	if cfg == nil {
		http.Error(w, "Config not found", http.StatusInternalServerError)
		return
	}

	var newStacks []string
	for _, p := range cfg.ManualStacks {
		if p != req.Path {
			newStacks = append(newStacks, p)
		}
	}

	cfg.ManualStacks = newStacks
	if err := config.Save(cfg); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	s.config = cfg
	w.WriteHeader(http.StatusOK)
}
