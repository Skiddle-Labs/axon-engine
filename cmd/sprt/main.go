package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// SPRTConfig holds the parameters for the Sequential Probability Ratio Test.
type SPRTConfig struct {
	Elo0  float64 // Null Hypothesis H0 (e.g., 0 Elo gain)
	Elo1  float64 // Alternative Hypothesis H1 (e.g., 5 Elo gain)
	Alpha float64 // Type I error probability (false positive)
	Beta  float64 // Type II error probability (false negative)
}

// Result tracks the game outcomes.
type Result struct {
	Wins   int
	Losses int
	Draws  int
}

var (
	engineNew = flag.String("new", "./axon", "Path to the candidate engine binary")
	engineOld = flag.String("old", "./axon-master", "Path to the reference engine binary")
	tc        = flag.String("tc", "10+0.1", "Time control (e.g., 10+0.1)")
	threads   = flag.Int("concurrency", 4, "Number of games to run in parallel")
	book      = flag.String("book", "procrasti.bin", "Path to the opening book")
	elo0      = flag.Float64("elo0", 0.0, "SPRT H0 Elo bound")
	elo1      = flag.Float64("elo1", 5.0, "SPRT H1 Elo bound")
	alpha     = flag.Float64("alpha", 0.05, "SPRT Alpha")
	beta      = flag.Float64("beta", 0.05, "SPRT Beta")
)

func main() {
	flag.Parse()

	cfg := SPRTConfig{
		Elo0:  *elo0,
		Elo1:  *elo1,
		Alpha: *alpha,
		Beta:  *beta,
	}

	fmt.Printf("Axon SPRT Controller\n")
	fmt.Printf("Candidate: %s\n", *engineNew)
	fmt.Printf("Reference: %s\n", *engineOld)
	fmt.Printf("SPRT: Elo0=%.1f, Elo1=%.1f, Alpha=%.2f, Beta=%.2f\n", cfg.Elo0, cfg.Elo1, cfg.Alpha, cfg.Beta)
	fmt.Printf("TC: %s | Concurrency: %d\n\n", *tc, *threads)

	runMatch(cfg)
}

func runMatch(cfg SPRTConfig) {
	// Build cutechess-cli command
	args := []string{
		"-engine", "name=Candidate", "cmd=" + *engineNew,
		"-engine", "name=Reference", "cmd=" + *engineOld,
		"-each", "proto=uci", "tc=" + *tc,
		"-concurrency", strconv.Itoa(*threads),
		"-repeat",
		"-recover",
		"-games", "20000", // High limit, SPRT will stop it
		"-pgnout", "sprt_games.pgn",
	}

	if *book != "" {
		if _, err := os.Stat(*book); err == nil {
			args = append(args, "-openings", "file="+*book, "format=polyglot", "order=random")
		} else {
			fmt.Printf("Warning: Opening book %s not found, skipping...\n", *book)
		}
	}

	cmd := exec.Command("cutechess-cli", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting cutechess-cli: %v\n", err)
		fmt.Println("Make sure 'cutechess-cli' is installed and in your PATH.")
		return
	}

	scanner := bufio.NewScanner(stdout)
	result := Result{}

	// Regex to parse cutechess-cli output: Finished game 1 (Candidate vs Reference): 1-0 {White mates}
	// Or: Finished game 2 (Reference vs Candidate): 1/2-1/2 {Draw by repetition}
	re := regexp.MustCompile(`Finished game \d+ \(([^)]+)\): ([0-9/.-]+)`)

	la := math.Log(*beta / (1.0 - *alpha))
	lb := math.Log((1.0 - *beta) / *alpha)

	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		players := matches[1]
		outcome := matches[2]

		// Determine if Candidate (New) won, lost, or drew
		if outcome == "1/2-1/2" {
			result.Draws++
		} else {
			candidateIsWhite := strings.HasPrefix(players, "Candidate")
			if outcome == "1-0" {
				if candidateIsWhite {
					result.Wins++
				} else {
					result.Losses++
				}
			} else if outcome == "0-1" {
				if candidateIsWhite {
					result.Losses++
				} else {
					result.Wins++
				}
			}
		}

		total := result.Wins + result.Losses + result.Draws
		if total == 0 {
			continue
		}

		llr := calculateLLR(result, cfg)
		elo, eloErr := calculateElo(result)

		fmt.Printf("\rGames: %d | W: %d, L: %d, D: %d | Elo: %.1f +/- %.1f | LLR: %.2f [%.2f, %.2f]",
			total, result.Wins, result.Losses, result.Draws, elo, eloErr, llr, la, lb)

		if llr >= lb {
			fmt.Printf("\n\nResult: ACCEPT (H1 passed)\n")
			cmd.Process.Kill()
			break
		} else if llr <= la {
			fmt.Printf("\n\nResult: REJECT (H0 failed)\n")
			cmd.Process.Kill()
			break
		}
	}

	cmd.Wait()
}

// calculateLLR computes the Log-Likelihood Ratio for the SPRT test.
func calculateLLR(r Result, cfg SPRTConfig) float64 {
	n := float64(r.Wins + r.Losses + r.Draws)
	w := float64(r.Wins) / n
	l := float64(r.Losses) / n
	d := float64(r.Draws) / n

	// Probabilities for H0 and H1
	p0 := eloToProb(cfg.Elo0)
	p1 := eloToProb(cfg.Elo1)

	// Log-likelihood for WDL model (Simplified)
	// We use the variance-based approximation for LLR
	mu0 := p0
	mu1 := p1

	// Variance of a single game result (Win=1, Draw=0.5, Loss=0)
	// Var(X) = E[X^2] - (E[X])^2
	mu := (w*1.0 + d*0.5 + l*0.0)
	varX := (w*1.0 + d*0.25 + l*0.0) - mu*mu

	if varX <= 0 {
		return 0
	}

	return n * (mu1 - mu0) * (2*mu - mu0 - mu1) / (2 * varX)
}

func eloToProb(elo float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, -elo/400.0))
}

func calculateElo(r Result) (float64, float64) {
	n := float64(r.Wins + r.Losses + r.Draws)
	w := float64(r.Wins) / n
	l := float64(r.Losses) / n
	d := float64(r.Draws) / n

	mu := w + 0.5*d
	elo := -400.0 * math.Log10(1.0/mu-1.0)

	// Error margin (95% confidence)
	devW := w * math.Pow(1.0-mu, 2)
	devL := l * math.Pow(0.0-mu, 2)
	devD := d * math.Pow(0.5-mu, 2)
	stdDev := math.Sqrt(devW + devL + devD)
	margin := 1.96 * stdDev / math.Sqrt(n)

	// Convert margin to Elo
	// Derivative of Elo(mu) = 400 / (ln(10) * mu * (1 - mu))
	eloErr := margin * (400.0 / (math.Log(10) * mu * (1.0 - mu)))

	return elo, eloErr
}
