package scenario

import "fmt"

// Stop representa una parada en la ruta
type Stop struct {
	ID       int     // ID de la parada
	Name     string  // Nombre de la parada
	Position float64 // Posición en la ruta (0.0 a 1.0)
}

// Route representa una ruta lineal con paradas
type Route struct {
	Name     string  // Nombre de la ruta
	Length   float64 // Longitud total en km
	Stops    []Stop  // Paradas en la ruta
	StartLat float64 // Latitud inicial
	StartLon float64 // Longitud inicial
	EndLat   float64 // Latitud final
	EndLon   float64 // Longitud final
}

// NewDefaultRoute crea una ruta de ejemplo
func NewDefaultRoute() *Route {
	return &Route{
		Name:     "Ruta 5 - Centro",
		Length:   10.0, // 10 km
		StartLat: 19.4326,
		StartLon: -99.1332,
		EndLat:   19.4426,  // ~1.1 km al norte
		EndLon:   -99.1232, // ~1.1 km al este
		Stops: []Stop{
			{ID: 1, Name: "Terminal Sur", Position: 0.0},
			{ID: 2, Name: "Centro Comercial", Position: 0.25},
			{ID: 3, Name: "Hospital General", Position: 0.5},
			{ID: 4, Name: "Universidad", Position: 0.75},
			{ID: 5, Name: "Terminal Norte", Position: 1.0},
		},
	}
}

// GetPositionAtProgress calcula lat/lon según el progreso en la ruta (0.0 a 1.0)
func (r *Route) GetPositionAtProgress(progress float64) (lat, lon float64) {
	// Clamp progress entre 0 y 1
	if progress < 0.0 {
		progress = 0.0
	}
	if progress > 1.0 {
		progress = 1.0
	}

	// Interpolación lineal
	lat = r.StartLat + (r.EndLat-r.StartLat)*progress
	lon = r.StartLon + (r.EndLon-r.StartLon)*progress

	return lat, lon
}

// GetNearestStop retorna la parada más cercana al progreso actual
func (r *Route) GetNearestStop(progress float64) *Stop {
	if len(r.Stops) == 0 {
		return nil
	}

	// Encontrar la parada más cercana
	nearestStop := &r.Stops[0]
	minDistance := abs(progress - nearestStop.Position)

	for i := range r.Stops {
		distance := abs(progress - r.Stops[i].Position)
		if distance < minDistance {
			minDistance = distance
			nearestStop = &r.Stops[i]
		}
	}

	return nearestStop
}

// GetNextStop retorna la próxima parada desde el progreso actual
func (r *Route) GetNextStop(progress float64) *Stop {
	for i := range r.Stops {
		if r.Stops[i].Position > progress {
			return &r.Stops[i]
		}
	}
	return nil // Ya pasamos todas las paradas
}

// GetDistanceToStop calcula distancia en km a una parada
func (r *Route) GetDistanceToStop(progress float64, stop *Stop) float64 {
	if stop == nil {
		return 0.0
	}

	progressDiff := stop.Position - progress
	if progressDiff < 0 {
		progressDiff = 0
	}

	return progressDiff * r.Length
}

// String implementa fmt.Stringer
func (r *Route) String() string {
	return fmt.Sprintf("Ruta: %s (%.1f km, %d paradas)", r.Name, r.Length, len(r.Stops))
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
