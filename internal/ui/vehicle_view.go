package ui

import (
	"fmt"
	"image/color"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// VehicleView muestra la vista del vehículo
type VehicleView struct {
	config *config.Config
	route  *scenario.Route

	// Colores
	colorBackground color.Color
	colorRoute      color.Color
	colorStop       color.Color
	colorVehicle    color.Color
	colorText       color.Color
	colorPanelBg    color.Color
}

// NewVehicleView crea una nueva vista del vehículo
func NewVehicleView(cfg *config.Config, route *scenario.Route) *VehicleView {
	return &VehicleView{
		config:          cfg,
		route:           route,
		colorBackground: color.RGBA{20, 20, 30, 255},
		colorRoute:      color.RGBA{100, 100, 120, 255},
		colorStop:       color.RGBA{255, 200, 0, 255},
		colorVehicle:    color.RGBA{0, 200, 100, 255},
		colorText:       color.RGBA{255, 255, 255, 255},
		colorPanelBg:    color.RGBA{30, 30, 40, 200},
	}
}

// Draw dibuja la vista del vehículo
func (vv *VehicleView) Draw(screen *ebiten.Image, gpsData eventbus.GPSData, mpuData eventbus.MPUData, vehicleState eventbus.VehicleStateData, progress float64) {
	width := float32(vv.config.UI.Window.Width)
	height := float32(vv.config.UI.Window.Height)

	// Dibujar ruta lineal (parte superior)
	vv.drawRoute(screen, gpsData.Speed, progress)

	// Dibujar paneles de información
	vv.drawGPSPanel(screen, 20, 200, gpsData)
	vv.drawMPUPanel(screen, width-320, 200, mpuData)
	vv.drawDoorPanel(screen, 20, 360, vehicleState.DoorOpen, 0)
	vv.drawStatePanel(screen, 20, height-180, vehicleState)
}

// drawRoute dibuja la ruta lineal con paradas
func (vv *VehicleView) drawRoute(screen *ebiten.Image, currentSpeed float64, progress float64) {
	width := float32(vv.config.UI.Window.Width)

	// Posición de la ruta (centrada, parte superior)
	routeY := float32(100)
	routeStartX := float32(100)
	routeEndX := width - 100
	routeLength := routeEndX - routeStartX

	// Dibujar línea de ruta
	vector.StrokeLine(screen, routeStartX, routeY, routeEndX, routeY, 4, vv.colorRoute, false)

	// Dibujar paradas
	for _, stop := range vv.route.Stops {
		stopX := routeStartX + float32(stop.Position)*routeLength

		// Círculo de parada
		vector.DrawFilledCircle(screen, stopX, routeY, 8, vv.colorStop, false)

		// Nombre de parada
		ebitenutil.DebugPrintAt(screen, stop.Name, int(stopX-30), int(routeY+15))
	}

	// Dibujar vehículo (combi) en la ruta USANDO PROGRESS REAL
	vehicleX := routeStartX + routeLength*float32(progress)
	vector.DrawFilledCircle(screen, vehicleX, routeY, 12, vv.colorVehicle, false)

	// Emoji de combi (placeholder)
	ebitenutil.DebugPrintAt(screen, "🚌", int(vehicleX-8), int(routeY-25))

	// Barra de progreso debajo
	progressBarY := routeY + 40
	progressBarWidth := routeLength * float32(progress)
	vector.DrawFilledRect(screen, routeStartX, progressBarY, progressBarWidth, 10, vv.colorVehicle, false)
	vector.StrokeRect(screen, routeStartX, progressBarY, routeLength, 10, 2, vv.colorRoute, false)

	// Texto de progreso
	progressText := fmt.Sprintf("Progreso: %.0f%% | Velocidad: %.1f km/h", progress*100, currentSpeed)
	ebitenutil.DebugPrintAt(screen, progressText, int(routeStartX), int(progressBarY+20))
}

// drawGPSPanel dibuja panel de información GPS
func (vv *VehicleView) drawGPSPanel(screen *ebiten.Image, x, y float32, data eventbus.GPSData) {
	// Panel de fondo
	vector.DrawFilledRect(screen, x, y, 280, 140, vv.colorPanelBg, false)
	vector.StrokeRect(screen, x, y, 280, 140, 2, vv.colorRoute, false)

	// Título
	ebitenutil.DebugPrintAt(screen, "📍 GPS", int(x+10), int(y+10))

	// Información
	yOffset := int(y + 35)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Lat: %.6f°", data.Latitude), int(x+10), yOffset)
	yOffset += 20
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Lon: %.6f°", data.Longitude), int(x+10), yOffset)
	yOffset += 20
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Velocidad: %.1f km/h", data.Speed), int(x+10), yOffset)
	yOffset += 20
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Satélites: %d", data.Satellites), int(x+10), yOffset)
	yOffset += 20
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Fix: %d", data.FixQuality), int(x+10), yOffset)
}

// drawMPUPanel dibuja panel de información MPU6050
func (vv *VehicleView) drawMPUPanel(screen *ebiten.Image, x, y float32, data eventbus.MPUData) {
	// Panel de fondo
	vector.DrawFilledRect(screen, x, y, 280, 140, vv.colorPanelBg, false)
	vector.StrokeRect(screen, x, y, 280, 140, 2, vv.colorRoute, false)

	// Título
	ebitenutil.DebugPrintAt(screen, "⚡ MPU6050", int(x+10), int(y+10))

	// Información
	yOffset := int(y + 35)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Aceleración: %.2f m/s²", data.AccelSmooth), int(x+10), yOffset)
	yOffset += 20
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Giro: %.1f°/s", data.GyroZ), int(x+10), yOffset)
	yOffset += 20

	// Estados
	statusText := "Estado: "
	if data.IsAccelerating {
		statusText += "🟢 ACELERANDO"
	} else if data.IsBraking {
		statusText += "🔴 FRENANDO"
	} else if data.IsTurning {
		statusText += "🔵 GIRANDO"
	} else {
		statusText += "⚪ DETENIDO"
	}
	ebitenutil.DebugPrintAt(screen, statusText, int(x+10), yOffset)
}

// drawStatePanel dibuja panel de estado del vehículo
func (vv *VehicleView) drawStatePanel(screen *ebiten.Image, x, y float32, state eventbus.VehicleStateData) {
	width := float32(vv.config.UI.Window.Width)
	panelWidth := width - 40

	// Panel de fondo
	vector.DrawFilledRect(screen, x, y, panelWidth, 140, vv.colorPanelBg, false)
	vector.StrokeRect(screen, x, y, panelWidth, 140, 2, vv.colorRoute, false)

	// Título
	ebitenutil.DebugPrintAt(screen, "🚗 ESTADO DEL VEHÍCULO", int(x+10), int(y+10))

	// Estado principal (grande y destacado)
	yOffset := int(y + 40)
	stateEmoji := getStateEmoji(state.State)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s %s", stateEmoji, state.State), int(x+10), yOffset)

	// Información adicional
	yOffset += 30
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Velocidad: %.1f km/h | Aceleración: %.2f m/s² | Giro: %.1f°/s",
		state.Speed, state.Acceleration, state.TurnRate), int(x+10), yOffset)

	yOffset += 25
	gpsStatus := "❌ Sin GPS"
	if state.HasGPSFix {
		gpsStatus = fmt.Sprintf("✅ GPS Fix (Calidad: %d)", state.GPSQuality)
	}
	ebitenutil.DebugPrintAt(screen, gpsStatus, int(x+10), yOffset)
}

// drawDoorPanel dibuja panel de información de puerta
func (vv *VehicleView) drawDoorPanel(screen *ebiten.Image, x, y float32, doorOpen bool, doorDistance int) {
	// Panel de fondo
	vector.DrawFilledRect(screen, x, y, 280, 100, vv.colorPanelBg, false)
	vector.StrokeRect(screen, x, y, 280, 100, 2, vv.colorRoute, false)

	// Título
	ebitenutil.DebugPrintAt(screen, "🚪 PUERTA", int(x+10), int(y+10))

	// Estado de puerta
	yOffset := int(y + 35)
	doorStatus := "🔴 CERRADA"
	if doorOpen {
		doorStatus = "🟢 ABIERTA"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Estado: %s", doorStatus), int(x+10), yOffset)

	yOffset += 20
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Distancia: %d mm", doorDistance), int(x+10), yOffset)
}

// getStateEmoji retorna emoji según el estado
func getStateEmoji(state string) string {
	switch state {
	case eventbus.VehicleDetenido, eventbus.VehicleDetenidoSinGPS:
		return "🛑"
	case eventbus.VehicleMovimientoConfirmado:
		return "🚀"
	case eventbus.VehicleGPSMovimiento:
		return "📡"
	case eventbus.VehicleMPUMovimiento, eventbus.VehicleMPUMovimientoSinGPS:
		return "⚡"
	default:
		return "🚌"
	}
}
