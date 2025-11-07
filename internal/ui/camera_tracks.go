package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TrackInfo representa información de un track activo
type TrackInfo struct {
	ID         int
	Confidence float64
	LastSeen   time.Time
}

// CameraTracks muestra los tracks activos de la cámara
type CameraTracks struct {
	mu sync.RWMutex

	x      float32
	y      float32
	width  float32
	height float32

	tracks     map[int]*TrackInfo
	nextID     int
	maxDisplay int
	maxAge     time.Duration

	lastCount  int
	lastUpdate time.Time

	// Colores
	colorBg     color.RGBA
	colorBorder color.RGBA
	colorTrack  color.RGBA
	colorText   color.RGBA
}

// NewCameraTracks crea un nuevo panel de tracks
func NewCameraTracks(x, y, width, height float32, maxDisplay int) *CameraTracks {
	return &CameraTracks{
		x:           x,
		y:           y,
		width:       width,
		height:      height,
		maxDisplay:  maxDisplay,
		tracks:      make(map[int]*TrackInfo),
		nextID:      1,
		maxAge:      3 * time.Second,
		colorBg:     color.RGBA{30, 30, 40, 255},
		colorBorder: color.RGBA{80, 80, 100, 255},
		colorTrack:  color.RGBA{100, 255, 150, 255},
		colorText:   color.RGBA{200, 200, 220, 255},
	}
}

// UpdateFromCameraData actualiza tracks desde CameraData
func (ct *CameraTracks) UpdateFromCameraData(data eventbus.CameraData) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	ct.lastUpdate = now

	// Usar DetectedPersons del CameraData
	detectedCount := data.DetectedPersons
	currentCount := len(ct.tracks)

	if detectedCount > currentCount {
		// Agregar nuevos tracks
		for i := currentCount; i < detectedCount; i++ {
			// Si hay información de tracks reales, usarla
			confidence := 0.85
			if i < len(data.Tracks) {
				confidence = data.Tracks[i].Confidence
			}

			ct.tracks[ct.nextID] = &TrackInfo{
				ID:         ct.nextID,
				Confidence: confidence,
				LastSeen:   now,
			}
			ct.nextID++
		}
	} else if detectedCount < currentCount {
		// Eliminar tracks más antiguos
		diff := currentCount - detectedCount
		for i := 0; i < diff; i++ {
			oldestID := -1
			var oldestTime time.Time

			for id, track := range ct.tracks {
				if oldestID == -1 || track.LastSeen.Before(oldestTime) {
					oldestID = id
					oldestTime = track.LastSeen
				}
			}

			if oldestID != -1 {
				delete(ct.tracks, oldestID)
			}
		}
	}

	// Actualizar LastSeen de todos los tracks
	for _, track := range ct.tracks {
		track.LastSeen = now
	}

	ct.cleanOldTracks(now)
}

// AddTrack agrega un nuevo track (para eventos de pasajeros)
func (ct *CameraTracks) AddTrack() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	ct.tracks[ct.nextID] = &TrackInfo{
		ID:         ct.nextID,
		Confidence: 0.80 + float64(ct.nextID%20)/100.0,
		LastSeen:   now,
	}
	ct.nextID++
	ct.lastUpdate = now
}

// RemoveOldestTrack elimina el track más antiguo
func (ct *CameraTracks) RemoveOldestTrack() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if len(ct.tracks) == 0 {
		return
	}

	oldestID := -1
	var oldestTime time.Time

	for id, track := range ct.tracks {
		if oldestID == -1 || track.LastSeen.Before(oldestTime) {
			oldestID = id
			oldestTime = track.LastSeen
		}
	}

	if oldestID != -1 {
		delete(ct.tracks, oldestID)
	}
}

// cleanOldTracks elimina tracks que no se han visto recientemente
func (ct *CameraTracks) cleanOldTracks(now time.Time) {
	for id, track := range ct.tracks {
		if now.Sub(track.LastSeen) > ct.maxAge {
			delete(ct.tracks, id)
		}
	}
}

// Clear limpia todos los tracks
func (ct *CameraTracks) Clear() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.tracks = make(map[int]*TrackInfo)
	ct.nextID = 1
	ct.lastCount = 0
}

// Draw dibuja el panel
func (ct *CameraTracks) Draw(screen *ebiten.Image) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	// Fondo
	vector.DrawFilledRect(screen, ct.x, ct.y, ct.width, ct.height, ct.colorBg, false)
	vector.StrokeRect(screen, ct.x, ct.y, ct.width, ct.height, 2, ct.colorBorder, false)

	// Título
	ebitenutil.DebugPrintAt(screen, "📷 DETECCIONES CÁMARA", int(ct.x+10), int(ct.y+10))

	trackCount := len(ct.tracks)

	if trackCount == 0 {
		ebitenutil.DebugPrintAt(screen, "Sin detecciones", int(ct.x+10), int(ct.y+35))
		return
	}

	// Dibujar tracks
	yOffset := int(ct.y + 35)
	lineHeight := 20
	count := 0

	for trackID := 1; trackID < ct.nextID && count < ct.maxDisplay; trackID++ {
		track, exists := ct.tracks[trackID]
		if !exists {
			continue
		}

		confidencePercent := int(track.Confidence * 100)
		trackText := fmt.Sprintf("Track #%d: 👤 (%d%%)", track.ID, confidencePercent)
		ebitenutil.DebugPrintAt(screen, trackText, int(ct.x+10), yOffset)

		yOffset += lineHeight
		count++
	}

	if trackCount > ct.maxDisplay {
		moreText := fmt.Sprintf("...y %d más", trackCount-ct.maxDisplay)
		ebitenutil.DebugPrintAt(screen, moreText, int(ct.x+10), yOffset)
	}

	// Total
	totalText := fmt.Sprintf("Total: %d persona(s)", trackCount)
	ebitenutil.DebugPrintAt(screen, totalText, int(ct.x+10), int(ct.y+ct.height-25))
}
