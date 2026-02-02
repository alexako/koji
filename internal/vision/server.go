package vision

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// Server provides a web interface for face enrollment and management.
type Server struct {
	db       *FaceDB
	detector FaceDetector
	addr     string

	mu            sync.Mutex
	activeSession *EnrollmentSession
	sessionOwner  string
}

// NewServer creates a new enrollment web server.
func NewServer(addr string, db *FaceDB, detector FaceDetector) *Server {
	return &Server{
		db:       db,
		detector: detector,
		addr:     addr,
	}
}

// Start begins serving the web interface.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/people", s.handlePeople)
	mux.HandleFunc("/api/people/", s.handlePerson)
	mux.HandleFunc("/api/enroll/start", s.handleEnrollStart)
	mux.HandleFunc("/api/enroll/frame", s.handleEnrollFrame)
	mux.HandleFunc("/api/enroll/finish", s.handleEnrollFinish)
	mux.HandleFunc("/api/enroll/cancel", s.handleEnrollCancel)
	mux.HandleFunc("/api/status", s.handleStatus)

	// Serve static files (embedded or from disk)
	mux.HandleFunc("/", s.handleIndex)

	server := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}

// handleIndex serves the main enrollment page.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(indexHTML))
}

// handleStatus returns Koji's current face recognition status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := struct {
		HasOwner    bool   `json:"has_owner"`
		OwnerName   string `json:"owner_name,omitempty"`
		PeopleCount int    `json:"people_count"`
	}{
		HasOwner:    s.db.HasOwner(),
		PeopleCount: len(s.db.ListPeople()),
	}

	if owner := s.db.GetOwner(); owner != nil {
		status.OwnerName = owner.Name
	}

	writeJSON(w, status)
}

// handlePeople lists all enrolled people.
func (s *Server) handlePeople(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	people := s.db.ListPeople()

	// Strip embeddings from response (they're large and sensitive)
	type personSummary struct {
		ID           string       `json:"id"`
		Name         string       `json:"name"`
		Relationship Relationship `json:"relationship"`
		EnrolledAt   time.Time    `json:"enrolled_at"`
		LastSeenAt   time.Time    `json:"last_seen_at"`
		SeenCount    int          `json:"seen_count"`
	}

	summaries := make([]personSummary, len(people))
	for i, p := range people {
		summaries[i] = personSummary{
			ID:           p.ID,
			Name:         p.Name,
			Relationship: p.Relationship,
			EnrolledAt:   p.EnrolledAt,
			LastSeenAt:   p.LastSeenAt,
			SeenCount:    p.SeenCount,
		}
	}

	writeJSON(w, summaries)
}

// handlePerson handles individual person operations (GET, DELETE).
func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/people/"):]
	if id == "" {
		http.Error(w, "missing person ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		person := s.db.GetPerson(id)
		if person == nil {
			http.Error(w, "person not found", http.StatusNotFound)
			return
		}
		writeJSON(w, person)

	case http.MethodDelete:
		if err := s.db.RemovePerson(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleEnrollStart begins a new enrollment session.
func (s *Server) handleEnrollStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name         string       `json:"name"`
		Relationship Relationship `json:"relationship"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Relationship == "" {
		req.Relationship = RelationshipFriend // default
	}

	// Check if this is owner enrollment and owner already exists
	if req.Relationship == RelationshipOwner && s.db.HasOwner() {
		http.Error(w, "owner already enrolled", http.StatusConflict)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeSession != nil {
		http.Error(w, "enrollment already in progress", http.StatusConflict)
		return
	}

	s.activeSession = NewEnrollmentSession(s.detector, s.db, req.Name, req.Relationship)
	s.sessionOwner = r.RemoteAddr

	writeJSON(w, map[string]string{
		"status":  "started",
		"message": "Enrollment started. Send frames to /api/enroll/frame",
	})
}

// handleEnrollFrame processes a frame during enrollment.
func (s *Server) handleEnrollFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	session := s.activeSession
	s.mu.Unlock()

	if session == nil {
		http.Error(w, "no enrollment session active", http.StatusBadRequest)
		return
	}

	// Read image data from request body
	imageData, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024)) // 10MB max
	if err != nil {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status, err := session.AddFrame(ctx, imageData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, status)
}

// handleEnrollFinish completes the enrollment session.
func (s *Server) handleEnrollFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	session := s.activeSession
	s.mu.Unlock()

	if session == nil {
		http.Error(w, "no enrollment session active", http.StatusBadRequest)
		return
	}

	if !session.CanFinish() {
		http.Error(w, "not enough samples collected", http.StatusBadRequest)
		return
	}

	person, err := session.Finish()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.activeSession = nil
	s.sessionOwner = ""
	s.mu.Unlock()

	writeJSON(w, map[string]interface{}{
		"status": "complete",
		"person": map[string]interface{}{
			"id":           person.ID,
			"name":         person.Name,
			"relationship": person.Relationship,
		},
	})
}

// handleEnrollCancel aborts the enrollment session.
func (s *Server) handleEnrollCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if s.activeSession != nil {
		s.activeSession.Cancel()
		s.activeSession = nil
		s.sessionOwner = ""
	}
	s.mu.Unlock()

	writeJSON(w, map[string]string{"status": "cancelled"})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// indexHTML is the embedded enrollment web page.
const indexHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Koji - Face Enrollment</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background: #1a1a2e;
            color: #eee;
        }
        h1 { color: #00d9ff; }
        .card {
            background: #16213e;
            border-radius: 12px;
            padding: 20px;
            margin: 20px 0;
        }
        video, canvas {
            width: 100%;
            border-radius: 8px;
            background: #000;
        }
        button {
            background: #00d9ff;
            color: #1a1a2e;
            border: none;
            padding: 12px 24px;
            border-radius: 8px;
            font-size: 16px;
            cursor: pointer;
            margin: 5px;
        }
        button:hover { background: #00b8d9; }
        button:disabled { background: #444; color: #888; cursor: not-allowed; }
        button.danger { background: #ff4757; color: white; }
        input {
            width: 100%;
            padding: 12px;
            border-radius: 8px;
            border: 1px solid #333;
            background: #0f0f23;
            color: #eee;
            font-size: 16px;
            margin: 10px 0;
        }
        select {
            width: 100%;
            padding: 12px;
            border-radius: 8px;
            border: 1px solid #333;
            background: #0f0f23;
            color: #eee;
            font-size: 16px;
            margin: 10px 0;
        }
        .status {
            padding: 10px;
            border-radius: 8px;
            margin: 10px 0;
        }
        .status.info { background: #1e3a5f; }
        .status.success { background: #1e5f3a; }
        .status.error { background: #5f1e1e; }
        .progress {
            height: 8px;
            background: #333;
            border-radius: 4px;
            overflow: hidden;
            margin: 10px 0;
        }
        .progress-bar {
            height: 100%;
            background: #00d9ff;
            transition: width 0.3s;
        }
        .people-list { list-style: none; padding: 0; }
        .people-list li {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 10px;
            border-bottom: 1px solid #333;
        }
        .people-list .relationship {
            font-size: 12px;
            color: #888;
            text-transform: uppercase;
        }
        #camera-container { display: none; }
        #camera-container.active { display: block; }
    </style>
</head>
<body>
    <h1>Koji Face Enrollment</h1>

    <div class="card">
        <h2>Status</h2>
        <div id="status" class="status info">Loading...</div>
    </div>

    <div class="card">
        <h2>Enrolled People</h2>
        <ul id="people-list" class="people-list">
            <li>Loading...</li>
        </ul>
    </div>

    <div class="card">
        <h2>Add New Person</h2>
        <input type="text" id="name" placeholder="Name">
        <select id="relationship">
            <option value="owner">Owner (primary)</option>
            <option value="family">Family member</option>
            <option value="friend" selected>Friend</option>
        </select>
        <button id="start-btn" onclick="startEnrollment()">Start Enrollment</button>
    </div>

    <div id="camera-container" class="card">
        <h2>Camera</h2>
        <video id="video" autoplay playsinline></video>
        <canvas id="canvas" style="display:none"></canvas>
        <div class="progress">
            <div id="progress-bar" class="progress-bar" style="width: 0%"></div>
        </div>
        <div id="enroll-status" class="status info">Position your face in the camera</div>
        <button onclick="finishEnrollment()">Finish</button>
        <button class="danger" onclick="cancelEnrollment()">Cancel</button>
    </div>

    <script>
        let stream = null;
        let enrolling = false;
        let frameInterval = null;

        async function loadStatus() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                const el = document.getElementById('status');
                if (data.has_owner) {
                    el.textContent = 'Owner: ' + data.owner_name + ' | ' + data.people_count + ' people enrolled';
                    el.className = 'status success';
                } else {
                    el.textContent = 'No owner enrolled yet. Add yourself as owner!';
                    el.className = 'status info';
                }
            } catch (e) {
                document.getElementById('status').textContent = 'Error loading status';
                document.getElementById('status').className = 'status error';
            }
        }

        async function loadPeople() {
            try {
                const res = await fetch('/api/people');
                const people = await res.json();
                const list = document.getElementById('people-list');
                if (people.length === 0) {
                    list.innerHTML = '<li>No one enrolled yet</li>';
                    return;
                }
                list.innerHTML = people.map(p => 
                    '<li><div><strong>' + p.name + '</strong><br>' +
                    '<span class="relationship">' + p.relationship + '</span></div>' +
                    '<button class="danger" onclick="removePerson(\'' + p.id + '\')">Remove</button></li>'
                ).join('');
            } catch (e) {
                document.getElementById('people-list').innerHTML = '<li>Error loading people</li>';
            }
        }

        async function removePerson(id) {
            if (!confirm('Remove this person?')) return;
            await fetch('/api/people/' + id, { method: 'DELETE' });
            loadPeople();
            loadStatus();
        }

        async function startEnrollment() {
            const name = document.getElementById('name').value.trim();
            const relationship = document.getElementById('relationship').value;

            if (!name) {
                alert('Please enter a name');
                return;
            }

            try {
                const res = await fetch('/api/enroll/start', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name, relationship })
                });

                if (!res.ok) {
                    const text = await res.text();
                    alert('Error: ' + text);
                    return;
                }

                // Start camera
                stream = await navigator.mediaDevices.getUserMedia({ 
                    video: { facingMode: 'user', width: 640, height: 480 } 
                });
                document.getElementById('video').srcObject = stream;
                document.getElementById('camera-container').classList.add('active');
                document.getElementById('start-btn').disabled = true;

                enrolling = true;
                frameInterval = setInterval(sendFrame, 500); // 2 fps

            } catch (e) {
                alert('Error: ' + e.message);
            }
        }

        async function sendFrame() {
            if (!enrolling) return;

            const video = document.getElementById('video');
            const canvas = document.getElementById('canvas');
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;
            canvas.getContext('2d').drawImage(video, 0, 0);

            try {
                const blob = await new Promise(r => canvas.toBlob(r, 'image/jpeg', 0.8));
                const res = await fetch('/api/enroll/frame', {
                    method: 'POST',
                    body: blob
                });
                const status = await res.json();

                document.getElementById('enroll-status').textContent = status.message;
                const progress = (status.samples_collected / status.samples_needed) * 100;
                document.getElementById('progress-bar').style.width = Math.min(progress, 100) + '%';

                if (status.is_complete) {
                    finishEnrollment();
                }
            } catch (e) {
                console.error('Frame error:', e);
            }
        }

        async function finishEnrollment() {
            enrolling = false;
            clearInterval(frameInterval);

            try {
                const res = await fetch('/api/enroll/finish', { method: 'POST' });
                if (res.ok) {
                    alert('Enrollment complete!');
                }
            } catch (e) {
                console.error('Finish error:', e);
            }

            cleanup();
        }

        async function cancelEnrollment() {
            enrolling = false;
            clearInterval(frameInterval);
            await fetch('/api/enroll/cancel', { method: 'POST' });
            cleanup();
        }

        function cleanup() {
            if (stream) {
                stream.getTracks().forEach(t => t.stop());
                stream = null;
            }
            document.getElementById('camera-container').classList.remove('active');
            document.getElementById('start-btn').disabled = false;
            document.getElementById('name').value = '';
            document.getElementById('progress-bar').style.width = '0%';
            loadPeople();
            loadStatus();
        }

        // Initial load
        loadStatus();
        loadPeople();
    </script>
</body>
</html>
`
