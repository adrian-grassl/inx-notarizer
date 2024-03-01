package notarizer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/adrian-grassl/inx-notarizer/pkg/hdwallet"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/builder"
	"github.com/iotaledger/iota.go/v3/nodeclient"
	"github.com/labstack/echo/v4"
)

type UTXOOutput struct {
	OutputID iotago.OutputID
	Output   *iotago.BasicOutput
}

type WalletObject struct {
	Bech32Address  string
	Ed25519Address *iotago.Ed25519Address
	AddressSigner  iotago.AddressSigner
}

const (
	inxRequestTimeout             = 5 * time.Second
	indexerPluginAvailableTimeout = 30 * time.Second
)

func createNotarization(c echo.Context) error {
	// Extract the hash parameter from the request path
	hash := c.Param("hash")
	Component.LogInfof("Received hash for notarization: %s", hash)

	protoParas := deps.NodeBridge.ProtocolParameters()

	// Prepare wallet address and signer
	walletObject, err := prepWallet(protoParas)
	if err != nil {
		Component.LogErrorf("Error preparing wallet: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error preparing wallet")
	}

	// Prepare outputs to be consumed.
	unspentOutputs, err := prepInputs(walletObject.Bech32Address)
	if err != nil {
		Component.LogErrorf("Error preparing inputs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error preparing inputs")
	}

	// Prepare transaction payload including the notarization hash.
	txPayload, err := prepTxPayload(protoParas, unspentOutputs, walletObject.Ed25519Address, walletObject.AddressSigner, hash)
	if err != nil {
		Component.LogErrorf("Error preparing transaction payload: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error preparing transaction payload")
	}

	// Prepare and send the block with the notarization transaction.
	hexBlockId, err := prepAndSendBlock(c, protoParas, txPayload)
	if err != nil {
		Component.LogErrorf("Error sending block: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error sending block")
	}
	Component.LogInfof("Block attached with ID: %v", hexBlockId)

	// Return success response with block ID.
	return c.JSON(http.StatusOK, map[string]string{"blockId": hexBlockId})
}

// loadEnvVariable loads mnemonic phrases from the given environment variable.
func loadEnvVariable(name string) ([]string, error) {
	keys, exists := os.LookupEnv(name)
	if !exists {
		return nil, fmt.Errorf("environment variable '%s' not set", name)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("environment variable '%s' not set", name)
	}

	phrases := strings.Split(keys, " ")
	return phrases, nil
}

// prepWallet prepares the wallet for transactions by loading the mnemonic and creating a wallet object.
func prepWallet(protoParas *iotago.ProtocolParameters) (*WalletObject, error) {
	mnemonic, err := loadEnvVariable("MNEMONIC")
	if err != nil {
		return nil, err
	}
	Component.LogDebugf("Mnemonic loaded successfully")

	wallet, err := hdwallet.NewHDWallet(protoParas, mnemonic, "", 0, false)
	if err != nil {
		return nil, fmt.Errorf("creating wallet failed, err: %s", err)
	}
	Component.LogDebugf("Wallet created successfully")

	address, signer, err := wallet.Ed25519AddressAndSigner(0)
	if err != nil {
		return nil, fmt.Errorf("deriving ed25519 address and signer failed, err: %s", err)
	}
	Component.LogDebugf("Address and signer derived successfully")

	bech32 := address.Bech32("tst")
	Component.LogDebugf("Bech32 Address: %v, %T", bech32, bech32)

	return &WalletObject{
		Bech32Address:  bech32,
		Ed25519Address: address,
		AddressSigner:  signer,
	}, nil
}

// prepInputs prepares the inputs for the transaction by fetching UTXO outputs for the wallet address.
func prepInputs(bech32 string) ([]UTXOOutput, error) {
	ctxIndexer, cancelIndexer := context.WithTimeout(context.Background(), indexerPluginAvailableTimeout)
	defer cancelIndexer()

	ctxRequest, cancelRequest := context.WithTimeout(context.Background(), inxRequestTimeout)
	defer cancelRequest()

	basicOutputsQuery := &nodeclient.BasicOutputsQuery{
		AddressBech32: bech32,
	}
	Component.LogInfof("Fetching UTXO outputs for address: %s", bech32)

	indexer, err := deps.NodeBridge.Indexer(ctxIndexer)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexer client: %v", err)
	}

	indexerResultSet, err := indexer.Outputs(ctxRequest, basicOutputsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch outputs from indexer: %v", err)
	}

	var unspentOutputs []UTXOOutput
	for indexerResultSet.Next() {
		outputs, err := indexerResultSet.Outputs()
		if err != nil {
			return nil, fmt.Errorf("failed to get outputs from indexer result set: %v", err)
		}

		for i, output := range outputs {
			if basicOutput, ok := output.(*iotago.BasicOutput); ok && basicOutput.FeatureSet().MetadataFeature() == nil {
				ingoingHexOutputId := iotago.HexOutputIDs{indexerResultSet.Response.Items[i]}
				ingoingOutputIds, err := ingoingHexOutputId.OutputIDs()
				if err != nil {
					panic(err)
				}

				unspentOutputs = append(unspentOutputs, UTXOOutput{
					OutputID: ingoingOutputIds[0],
					Output:   basicOutput,
				})
			}
		}
	}
	if indexerResultSet.Error != nil {
		return nil, fmt.Errorf("indexer result set error: %v", indexerResultSet.Error)
	}

	return unspentOutputs, nil
}

// prepTxPayload prepares the transaction payload by incorporating the notarization hash and creating outputs.
func prepTxPayload(protoParas *iotago.ProtocolParameters, unspentOutputs []UTXOOutput, address *iotago.Ed25519Address, signer iotago.AddressSigner, hash string) (*iotago.Transaction, error) {
	txBuilder := builder.NewTransactionBuilder(protoParas.NetworkID())
	Component.LogInfof("Building transaction with network ID: %v", protoParas.NetworkID)

	var totalDeposit uint64 = 0
	for _, unspentOutput := range unspentOutputs {
		txBuilder.AddInput(&builder.TxInput{
			UnlockTarget: address,
			InputID:      unspentOutput.OutputID,
			Input:        unspentOutput.Output,
		})
		totalDeposit += unspentOutput.Output.Deposit()
	}

	// Add a basic output with the notarization hash as metadata.
	txBuilder.AddOutput(&iotago.BasicOutput{
		Amount: uint64(1000000), // Example amount for notarization, adjust as needed.
		Conditions: iotago.UnlockConditions{
			&iotago.AddressUnlockCondition{Address: address},
		},
		Features: iotago.Features{
			&iotago.MetadataFeature{Data: []byte(hash)},
		},
	})

	// Add a basic output for the token remainder to send back to the sender.
	if remainder := totalDeposit - 1000000; remainder > 0 {
		txBuilder.AddOutput(&iotago.BasicOutput{
			Amount: remainder,
			Conditions: iotago.UnlockConditions{
				&iotago.AddressUnlockCondition{Address: address},
			},
		})
	}

	txPayload, err := txBuilder.Build(protoParas, signer)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction payload: %v", err)
	}

	return txPayload, nil
}

// prepAndSendBlock prepares and submits a block with the transaction payload.
func prepAndSendBlock(c echo.Context, protoParas *iotago.ProtocolParameters, txPayload *iotago.Transaction) (string, error) {
	ctx := c.Request().Context()

	transactionID, err := txPayload.ID()
	if err != nil {
		return "", fmt.Errorf("failed to get transaction ID: %v", err)
	}
	Component.LogInfof("Transaction ID: %s", transactionID.ToHex())

	inxNodeClient := deps.NodeBridge.INXNodeClient()

	tipsResponse, err := inxNodeClient.Tips(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tips: %v", err)
	}

	tips, err := tipsResponse.Tips()
	if err != nil {
		return "", fmt.Errorf("failed to parse tips from response: %v", err)
	}

	block, err := builder.NewBlockBuilder().
		ProtocolVersion(protoParas.Version).
		Parents(tips).
		Payload(txPayload).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build block: %v", err)
	}

	blockId, err := inxNodeClient.SubmitBlock(ctx, block, protoParas)
	if err != nil {
		return "", fmt.Errorf("failed to submit block: %v", err)
	}

	return blockId.ToHex(), nil
}
