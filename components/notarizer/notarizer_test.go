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
