package notarizer

import (
	"os"
	"testing"

	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/stretchr/testify/assert"
)

func TestLoadEnvVariable(t *testing.T) {
	t.Run("Environment variable set correctly", func(t *testing.T) {
		// Setup
		envVarName := "TEST_MNEMONIC"
		expectedValue := []string{"word1", "word2", "word3"}
		os.Setenv(envVarName, "word1 word2 word3")
		defer os.Unsetenv(envVarName) // Cleanup

		// Execute
		result, err := loadEnvVariable(envVarName)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, result)
	})

	t.Run("Environment variable not set", func(t *testing.T) {
		// Setup
		envVarName := "NONEXISTENT_ENV_VAR"

		// Execute
		result, err := loadEnvVariable(envVarName)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.EqualError(t, err, "environment variable 'NONEXISTENT_ENV_VAR' not set")
	})

	t.Run("Environment variable set but empty", func(t *testing.T) {
		// Setup
		envVarName := "EMPTY_ENV_VAR"
		os.Setenv(envVarName, "")
		defer os.Unsetenv(envVarName) // Cleanup

		// Execute
		result, err := loadEnvVariable(envVarName)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.EqualError(t, err, "environment variable 'EMPTY_ENV_VAR' not set")
	})
}

func TestPrepWallet(t *testing.T) {
	t.Run("Valid mnemonic seed phrase", func(t *testing.T) {
		// Setup
		mnemonic := mockMnemonic()
		protoParas := mockProtocolParameters()
		expectedValue := "tst1qzguhtxyuhgp4aklfkyd5ek3wtnta649pqvccrep95kesjf5kxuzvexrv6n"

		// Execute
		walletObject, err := prepWallet(protoParas, mnemonic)
		t.Logf("walletObject: %v", walletObject)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, walletObject.Bech32Address)
	})
}

func TestFilterOutputs(t *testing.T) {
	t.Run("Several Output Types, with and without metadata", func(t *testing.T) {
		// Setup
		outputs := mockUnfilteredOutputs()
		expectedValue := mockFilteredOutputs()

		// Execute
		filterOutputs, err := filterOutputs(outputs)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, filterOutputs)
	})
}

func mockProtocolParameters() *iotago.ProtocolParameters {
	return &iotago.ProtocolParameters{
		Version:       2,
		NetworkName:   "private_tangle1",
		Bech32HRP:     "tst",
		MinPoWScore:   0,
		BelowMaxDepth: 15,
		RentStructure: iotago.RentStructure{
			VByteCost:    500,
			VBFactorData: 1,
			VBFactorKey:  10,
		},
		TokenSupply: 2779530283277761,
	}
}

func mockMnemonic() []string {
	return []string{
		"pass", "improve", "fitness", "dress", "range",
		"orphan", "mass", "story", "tree", "meat",
		"evidence", "ostrich", "render", "shock", "ancient",
		"minute", "hip", "feature", "split", "rigid",
		"way", "figure", "wasp", "property",
	}
}

func mockUnfilteredOutputs() []UTXOOutput {
	// Mock outputs for testing
	return []UTXOOutput{

		// Valid Output: Basic Output without MetadataFeature
		{
			OutputID: [iotago.OutputIDLength]byte{4, 5, 6}, // Example ID
			Output: &iotago.BasicOutput{
				Amount: 2000,
				Conditions: iotago.UnlockConditions{
					&iotago.AddressUnlockCondition{Address: &iotago.Ed25519Address{}},
				},
			},
		},

		// Invalid Output: Basic Output with MetadataFeature
		{
			OutputID: [iotago.OutputIDLength]byte{1, 2, 3}, // Example ID
			Output: &iotago.BasicOutput{
				Amount: 1000,
				Conditions: iotago.UnlockConditions{
					&iotago.AddressUnlockCondition{Address: &iotago.Ed25519Address{}},
				},
				Features: iotago.Features{
					&iotago.MetadataFeature{Data: []byte("metadata1")},
				},
			},
		},

		// Invalid Output: Alias Output
		{
			OutputID: [iotago.OutputIDLength]byte{1, 2, 3}, // Example ID
			Output: &iotago.AliasOutput{
				Amount: 1000,
				Conditions: iotago.UnlockConditions{
					&iotago.AddressUnlockCondition{Address: &iotago.Ed25519Address{}},
				},
			},
		},
	}
}

func mockFilteredOutputs() []BasicOutput {
	return []BasicOutput{
		{
			OutputID: [iotago.OutputIDLength]byte{4, 5, 6},
			Output: &iotago.BasicOutput{
				Amount: 2000,
				Conditions: iotago.UnlockConditions{
					&iotago.AddressUnlockCondition{Address: &iotago.Ed25519Address{}},
				},
			},
		},
	}
}
