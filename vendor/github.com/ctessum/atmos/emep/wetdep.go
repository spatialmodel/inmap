package emep

// Calculate wet deposition based on formulas at
// www.emep.int/UniDoc/node12.html.
// Inputs are fraction of grid cell covered by clouds (cloudFrac),
// rain mixing ratio (qrain), air density (ρair [kg/m3]),
// and fall distance (Δz [m]).
// Outputs are wet deposition rates for PM2.5, SO2, and other gases
// (wdParticle, wdSO2, and wdOtherGas [1/s]).
func WetDeposition(cloudFrac, qrain, ρair, Δz float64) (
	wdParticle, wdSO2, wdOtherGas float64) {
	const A = 5.2          // m3 kg-1 s-1; Empirical coefficient
	const E = 0.1          // size-dependent collection efficiency of aerosols by the raindrops
	const wSubSO2 = 0.15   // sub-cloud scavanging ratio
	const wSubOther = 0.5  // sub-cloud scavanging ratio
	const wInSO2 = 0.3     // in-cloud scavanging ratio
	const wInParticle = 1. // in-cloud scavanging ratio
	const wInOther = 1.4   // in-cloud scavanging ratio
	const ρwater = 1000.   // kg/m3
	const Vdr = 5.         // m/s

	// precalculated constant combinations
	const AE = A * E
	const wSubSO2VdrPerρwater = wSubSO2 * Vdr / ρwater
	const wSubOtherVdrPerρwater = wSubOther * Vdr / ρwater
	const wInSO2VdrPerρwater = wInSO2 * Vdr / ρwater
	const wInParticleVdrPerρwater = wInParticle * Vdr / ρwater
	const wInOtherVdrPerρwater = wInOther * Vdr / ρwater

	// wdParticle (subcloud) = A * P / Vdr * E; P = QRAIN * Vdr * ρgas =>
	//		wdParticle = A * QRAIN * ρgas * E
	// wdGas (subcloud) = wSub * P / Δz / ρwater =
	//		wSub * QRAIN * Vdr * ρgas / Δz / ρwater
	// wd (in-cloud) = wIn * P / Δz / ρwater =
	//		wIn * QRAIN * Vdr * ρgas / Δz / ρwater

	wdParticle = qrain * ρair * (AE +
		cloudFrac*(wInParticleVdrPerρwater/Δz))
	wdSO2 = (wSubSO2VdrPerρwater + cloudFrac*wSubSO2VdrPerρwater) *
		qrain * ρair / Δz
	wdOtherGas = (wSubOtherVdrPerρwater + cloudFrac*wSubOtherVdrPerρwater) *
		qrain * ρair / Δz
	return
}
