package ecdsa

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/nanlour/da/vdf_go"
)

// TestSign verifies that signatures created with Sign can be verified
func TestSign(t *testing.T) {
	// Generate a key pair
	privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Test message
	message := []byte("Hello, world!")

	// Sign the message
	signature, err := Sign(privateKey, message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Verify the signature
	valid := Verify(&privateKey.PublicKey, message, signature)
	if !valid {
		t.Errorf("Signature verification failed")
	}

	// Modify the message and verify it should fail
	modifiedMessage := []byte("Hello, World!")
	valid = Verify(&privateKey.PublicKey, modifiedMessage, signature)
	if valid {
		t.Errorf("Verification succeeded with modified message, expected failure")
	}

	// Modify the signature and verify it should fail
	modifiedSignature := make([]byte, len(signature))
	copy(modifiedSignature, signature)
	modifiedSignature[0] ^= 0x01 // Flip a bit
	valid = Verify(&privateKey.PublicKey, message, modifiedSignature)
	if valid {
		t.Errorf("Verification succeeded with tampered signature, expected failure")
	}
}

// TestSignConsistency ensures that signing the same message multiple times produces valid signatures
func TestSignConsistency(t *testing.T) {
	privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message := []byte("Consistent message")

	// Sign multiple times
	for i := 0; i < 5; i++ {
		signature, err := Sign(privateKey, message)
		if err != nil {
			t.Fatalf("Failed to sign message in iteration %d: %v", i, err)
		}

		// Verify each signature
		if !Verify(&privateKey.PublicKey, message, signature) {
			t.Errorf("Signature verification failed in iteration %d", i)
		}
	}
}

// TestSignDifferentMessages verifies that different messages produce different signatures
func TestSignDifferentMessages(t *testing.T) {
	privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message1 := []byte("First message")
	message2 := []byte("Second message")

	signature1, err := Sign(privateKey, message1)
	if err != nil {
		t.Fatalf("Failed to sign first message: %v", err)
	}

	signature2, err := Sign(privateKey, message2)
	if err != nil {
		t.Fatalf("Failed to sign second message: %v", err)
	}

	// Signatures should be different
	if bytes.Equal(signature1, signature2) {
		t.Errorf("Different messages produced identical signatures")
	}

	// Cross verification should fail
	if Verify(&privateKey.PublicKey, message1, signature2) {
		t.Errorf("Verification succeeded with wrong signature for message1")
	}

	if Verify(&privateKey.PublicKey, message2, signature1) {
		t.Errorf("Verification succeeded with wrong signature for message2")
	}
}

// TestDifficultySeed verifies that difficultySeed produces deterministic results
func TestDifficultySeed(t *testing.T) {
	epochHash := sha256.Sum256([]byte("test epoch hash"))
	height := uint64(12345)

	// Calculate seed twice
	seed1 := difficultySeed(&epochHash, height)
	seed2 := difficultySeed(&epochHash, height)

	// Seeds should be identical for same inputs
	if seed1 != seed2 {
		t.Errorf("difficultySeed not deterministic")
	}

	// Different height should produce different seed
	differentHeight := height + 1
	seed3 := difficultySeed(&epochHash, differentHeight)

	if seed1 == seed3 {
		t.Errorf("Seeds should differ with different heights")
	}
}

// TestDifficulty verifies basic properties of the Difficulty function
func TestDifficulty(t *testing.T) {
	signature := []byte("test signature")
	stakeSum := 1000.0
	stakeMine := 100.0
	miningDifficulty := uint64(10)

	// Calculate difficulty
	diff := Difficulty(signature, stakeSum, stakeMine, miningDifficulty)

	// Same inputs should give same difficulty
	diff2 := Difficulty(signature, stakeSum, stakeMine, miningDifficulty)
	if diff != diff2 {
		t.Errorf("Difficulty function not deterministic")
	}
}

// TestDifficultyStatistics runs statistical tests on the difficulty calculation
func TestDifficultyStatistics(t *testing.T) {
	// Set up parameters
	stakeSum := 1000.0
	stakeMine := 50.0
	miningDifficulty := uint64(10000)
	iterations := 10000

	var min, max, sum uint64
	min = ^uint64(0) // Max uint64 value

	// Generate a key pair for signing
	privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	epochHash := sha256.Sum256([]byte("epoch hash for testing"))

	// Run the test multiple times
	for i := 0; i < iterations; i++ {
		// Create a unique seed for each iteration
		height := uint64(i)
		seed := difficultySeed(&epochHash, height)

		// Sign the seed
		seedBytes := seed[:]
		signature, err := Sign(privateKey, seedBytes)
		if err != nil {
			t.Fatalf("Failed to sign seed in iteration %d: %v", i, err)
		}

		// Calculate difficulty
		diff := Difficulty(signature, stakeSum, stakeMine, miningDifficulty)

		// Update statistics
		if diff < min {
			min = diff
		}
		if diff > max {
			max = diff
		}
		sum += diff
	}

	// Calculate average
	avg := float64(sum) / float64(iterations)

	fmt.Printf("Difficulty Statistics (over %d iterations):\n", iterations)
	fmt.Printf("Min: %d\n", min)
	fmt.Printf("Max: %d\n", max)
	fmt.Printf("Avg: %.2f\n", avg)

	// Ensure we have a reasonable distribution
	if max == min {
		t.Errorf("No variation in difficulty values (min=max=%d)", min)
	}
}

func TestVDFBasics(t *testing.T) {
    // Create a test input
    input := sha256.Sum256([]byte("test input"))
    
    // Create a new VDF with low difficulty for quick testing
    difficulty := 100
    vdf := vdf_go.New(difficulty, input)
    
    // Execute the VDF
    stopChan := make(chan struct{})
    go vdf.Execute(stopChan)
    
    // Wait for the result
    var output [516]byte
    select {
    case output = <-vdf.GetOutputChannel():
        // Got the output
    case <-time.After(5 * time.Second):
        t.Fatalf("VDF execution timed out")
    }
    
    // Check that the VDF is marked as finished
    if !vdf.IsFinished() {
        t.Errorf("VDF should be marked as finished")
    }
    
    // Verify the proof
    if !vdf.Verify(output) {
        t.Errorf("VDF proof verification failed")
    }

	vdf_ := vdf_go.New(difficulty, input)
	if !vdf_.Verify(output) {
        t.Errorf("VDF proof verification failed with New VDF")
    }
}

// TestVDFVerification tests that VDF verification works correctly
func TestVDFVerification(t *testing.T) {
    // Create a test input
    input := sha256.Sum256([]byte("verification test"))
    
    // Create a new VDF with low difficulty for quick testing
    difficulty := 100
    vdf := vdf_go.New(difficulty, input)
    
    // Execute the VDF
    stopChan := make(chan struct{})
    go vdf.Execute(stopChan)
    
    // Wait for the result
    var validProof [516]byte
    select {
    case validProof = <-vdf.GetOutputChannel():
        // Got the output
    case <-time.After(5 * time.Second):
        t.Fatalf("VDF execution timed out")
    }
    
    // Test that valid proof passes verification
    if !vdf.Verify(validProof) {
        t.Errorf("Valid VDF proof verification failed")
    }
    
    // Create an invalid proof by modifying a few bytes
    invalidProof := validProof
    invalidProof[0] ^= 0xFF
    invalidProof[100] ^= 0xFF
    invalidProof[200] ^= 0xFF
    
    // Test that invalid proof fails verification
    if vdf.Verify(invalidProof) {
        t.Errorf("Invalid VDF proof verification should have failed")
    }
}

// TestVDFDifficultyImpact tests how different difficulty levels affect VDF execution time
func TestVDFDifficultyImpact(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping VDF difficulty impact test in short mode")
    }
    
    // Create a test input
    input := sha256.Sum256([]byte("difficulty test"))
    
    difficulties := []int{10, 100, 500}
    times := make([]time.Duration, len(difficulties))
    
    fmt.Println("VDF Difficulty Impact Test:")
    
    for i, difficulty := range difficulties {
        // Create a new VDF with the specified difficulty
        vdf := vdf_go.New(difficulty, input)
        
        // Start timing
        startTime := time.Now()
        
        // Execute the VDF
        stopChan := make(chan struct{})
        go vdf.Execute(stopChan)
        
        // Wait for the result
        select {
        case <-vdf.GetOutputChannel():
            // Got the output
            execTime := time.Since(startTime)
            times[i] = execTime
            fmt.Printf("  Difficulty %d: %v\n", difficulty, execTime)
        case <-time.After(30 * time.Second):
            t.Fatalf("VDF execution timed out for difficulty %d", difficulty)
        }
    }
    
    // Verify that higher difficulty correlates with longer execution time
    for i := 1; i < len(difficulties); i++ {
        if times[i] <= times[i-1] {
            t.Logf("Warning: Higher difficulty %d not slower than %d (%v <= %v)",
                difficulties[i], difficulties[i-1], times[i], times[i-1])
        }
    }
}

