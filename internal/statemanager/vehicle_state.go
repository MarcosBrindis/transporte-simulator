package statemanager

import (
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// VehicleStateCalculator calcula el estado del vehículo basado en GPS + MPU
type VehicleStateCalculator struct {
	movementThreshold float64 // Umbral de velocidad GPS (km/h)
}

// NewVehicleStateCalculator crea un nuevo calculador de estado
func NewVehicleStateCalculator(movementThreshold float64) *VehicleStateCalculator {
	return &VehicleStateCalculator{
		movementThreshold: movementThreshold,
	}
}

// Calculate determina el estado del vehículo basado en GPS y MPU
func (vsc *VehicleStateCalculator) Calculate(gpsData eventbus.GPSData, mpuData eventbus.MPUData) eventbus.VehicleStateData {
	// Determinar si hay movimiento según GPS
	gpsMoving := gpsData.Speed >= vsc.movementThreshold

	// Determinar si hay movimiento según MPU
	mpuDetecting := mpuData.IsAccelerating || mpuData.IsTurning

	// Determinar si hay fix GPS válido
	hasGPSFix := gpsData.FixQuality > 0 && gpsData.Satellites >= 4

	// Calcular estado híbrido
	state := vsc.determineState(gpsMoving, mpuDetecting, hasGPSFix)

	return eventbus.VehicleStateData{
		State:        state,
		Speed:        gpsData.Speed,
		Acceleration: mpuData.AccelSmooth,
		TurnRate:     mpuData.GyroZ,
		IsMoving:     gpsMoving || mpuDetecting,
		IsStopped:    !gpsMoving && !mpuDetecting,
		DoorOpen:     false, // Se actualizará cuando integremos VL53L0X
		HasGPSFix:    hasGPSFix,
		GPSQuality:   gpsData.FixQuality,
		Timestamp:    time.Now(),
	}
}

// determineState determina el estado del vehículo
func (vsc *VehicleStateCalculator) determineState(gpsMoving, mpuDetecting, hasGPSFix bool) string {
	if hasGPSFix {
		if gpsMoving && mpuDetecting {
			return eventbus.VehicleMovimientoConfirmado
		} else if gpsMoving {
			return eventbus.VehicleGPSMovimiento
		} else if mpuDetecting {
			return eventbus.VehicleMPUMovimiento
		} else {
			return eventbus.VehicleDetenido
		}
	} else {
		// Sin GPS fix
		if mpuDetecting {
			return eventbus.VehicleMPUMovimientoSinGPS
		} else {
			return eventbus.VehicleDetenidoSinGPS
		}
	}
}
