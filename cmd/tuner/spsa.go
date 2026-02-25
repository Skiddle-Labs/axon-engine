package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// SPSAConfig contains the hyper-parameters for the SPSA optimization algorithm.
type SPSAConfig struct {
	A     float64 // Scaling factor for the step size (a_k)
	C     float64 // Scaling factor for the perturbation size (c_k)
	Alpha float64 // Decay rate for the step size (standard: 0.602)
	Gamma float64 // Decay rate for the perturbation size (standard: 0.101)
}

// DefaultSPSAConfig returns a configuration with tuned defaults for chess engine parameters.
func DefaultSPSAConfig() SPSAConfig {
	return SPSAConfig{
		A:     50.0, // Initial step size scale
		C:     2.0,  // Initial perturbation scale
		Alpha: 0.602,
		Gamma: 0.101,
	}
}

// RunSPSA performs the Simultaneous Perturbation Stochastic Approximation optimization.
// SPSA is highly efficient for high-dimensional tuning as it only requires two MSE calculations
// per iteration to estimate the gradient for ALL parameters simultaneously.
func RunSPSA(entries []PrecomputedEntry, k float64, iterations int) {
	params, names := getTunableParams()
	cfg := DefaultSPSAConfig()

	// theta stores the continuous floating-point values of our parameters.
	// bestTheta tracks the best configuration found during the search.
	theta := make([]float64, len(params))
	bestTheta := make([]float64, len(params))
	for i, p := range params {
		theta[i] = float64(*p)
		bestTheta[i] = theta[i]
	}

	lossName := strings.ToUpper(*lossType)
	bestLoss := CalculateMSEParallel(entries, k)
	fmt.Printf("Initial %s: %.10f\n", lossName, bestLoss)
	fmt.Printf("Starting SPSA optimization for %d iterations...\n", iterations)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// A stability offset for the step size, usually a small percentage of total iterations.
	aOffset := float64(iterations) * 0.05
	if aOffset < 10 {
		aOffset = 10
	}

	for iter := 1; iter <= iterations; iter++ {
		// 1. Calculate gain sequences for this iteration.
		// ak is the step size; ck is the perturbation magnitude.
		ak := cfg.A / math.Pow(float64(iter)+aOffset, cfg.Alpha)
		ck := cfg.C / math.Pow(float64(iter), cfg.Gamma)

		// 2. Generate a random perturbation vector Delta (Bernoulli ±1).
		delta := make([]float64, len(theta))
		for i := range delta {
			if rng.Float64() < 0.5 {
				delta[i] = 1.0
			} else {
				delta[i] = -1.0
			}
		}

		// 3. Evaluate MSE at (theta + ck*Delta).
		for i, p := range params {
			*p = int(math.Round(theta[i] + ck*delta[i]))
		}
		plusLoss := CalculateMSEParallel(entries, k)

		// 4. Evaluate loss at (theta - ck*Delta).
		for i, p := range params {
			*p = int(math.Round(theta[i] - ck*delta[i]))
		}
		minusLoss := CalculateMSEParallel(entries, k)

		// 5. Update theta using the Simultaneous Perturbation gradient estimate.
		// Gradient g_k = (plusLoss - minusLoss) / (2 * ck * Delta)
		// Since Delta_i is ±1, 1/Delta_i is simply Delta_i.
		gradientMultiplier := ak * (plusLoss - minusLoss) / (2.0 * ck)
		for i := range theta {
			theta[i] -= gradientMultiplier * delta[i]
			// Sync the actual evaluation parameters with the rounded theta values.
			*params[i] = int(math.Round(theta[i]))
		}

		// 6. Progress reporting and live-saving.
		if iter%10 == 0 || iter == 1 {
			currentLoss := CalculateMSEParallel(entries, k)
			if currentLoss < bestLoss {
				bestLoss = currentLoss
				copy(bestTheta, theta)
				saveParams(*saveFile, params, names)
				fmt.Printf("Iteration %d | ak: %.6f | ck: %.6f | %s: %.10f (Improved!)\n", iter, ak, ck, lossName, currentLoss)
			} else if iter%100 == 0 {
				fmt.Printf("Iteration %d | %s: %.10f\n", iter, lossName, currentLoss)
			}
		}

		// Print key parameters periodically for visual feedback.
		if iter%500 == 0 {
			printParams(params, names)
		}
	}

	// Restore the engine to the best parameters found.
	for i, p := range params {
		*p = int(math.Round(bestTheta[i]))
	}

	fmt.Println("\nSPSA Optimization complete.")
	printParams(params, names)
	saveParams(*saveFile, params, names)
}
