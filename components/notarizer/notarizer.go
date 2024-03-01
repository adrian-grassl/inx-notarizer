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

func attachTx(c echo.Context) error {

	protoParas := deps.NodeBridge.ProtocolParameters()

	// Prepare wallet address and signer
	walletObject, err := prepWallet(protoParas)
	if err != nil {
		Component.LogErrorf("Error preparing wallet: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error preparing wallet")
	}

	// Prepare to be consumed outputs
	unspentOutputs := prepInputs(walletObject.Bech32Address)

	// Prepare transaction payload
	txPayload := prepTxPayload(protoParas, unspentOutputs, walletObject.Ed25519Address, walletObject.AddressSigner)

	// Prepare and send block
	hexBlockId := prepAndSendBlock(c, protoParas, txPayload)
	Component.LogInfof("blockId: %v, %T", hexBlockId, hexBlockId)

	// Return success response with block ID
	return c.JSON(http.StatusOK, map[string]string{"blockId": hexBlockId})
}

// loads Mnemonic phrases from the given environment variable.
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

func prepWallet(protoParas *iotago.ProtocolParameters) (*WalletObject, error) {
	mnemonic, err := loadEnvVariable("MNEMONIC")
	if err != nil {
		return nil, err
	}
	Component.LogDebugf("Mnemonic loaded successfully")

	var wallet *hdwallet.HDWallet
	if len(mnemonic) > 0 {
		// new HDWallet instance for address derivation
		wallet, err = hdwallet.NewHDWallet(protoParas, mnemonic, "", 0, false)
		if err != nil {
			return nil, fmt.Errorf("creating wallet failed, err: %s", err)
		}
		Component.LogDebugf("Wallet created successfully")
	}

	address, signer, err := wallet.Ed25519AddressAndSigner(0)
	if err != nil {
		return nil, fmt.Errorf("deriving ed25519 address and signer failed, err: %s", err)
	}
	Component.LogDebugf("Address: %v, %T", address, address)
	Component.LogDebugf("Signer: %v, %T", signer, signer)

	bech32 := address.Bech32("tst")
	Component.LogDebugf("Bech32 Address: %v, %T", bech32, bech32)

	return &WalletObject{
		Bech32Address:  bech32,
		Ed25519Address: address,
		AddressSigner:  signer,
	}, nil
}

func prepInputs(bech32 string) []UTXOOutput {

	ctxIndexer, cancelIndexer := context.WithTimeout(Component.Daemon().ContextStopped(), indexerPluginAvailableTimeout)
	defer cancelIndexer()

	ctxRequest, cancelRequest := context.WithTimeout(Component.Daemon().ContextStopped(), inxRequestTimeout)
	defer cancelRequest()

	basicOutputsQuery := &nodeclient.BasicOutputsQuery{
		AddressBech32: bech32,
	}
	Component.LogInfof("basicOutputsQuery: %v, %T", basicOutputsQuery, basicOutputsQuery)

	indexer, err := deps.NodeBridge.Indexer(ctxIndexer)
	if err != nil {
		Component.LogErrorfAndExit("Indexer client instance failed, err: %s", err)
	} else {
		Component.LogInfof("Indexer client instance loaded successfully")
	}

	indexerResultSet, err := indexer.Outputs(ctxRequest, basicOutputsQuery)
	if err != nil {
		panic(err)
	}
	Component.LogInfof("indexerResultSet: %v, %T", indexerResultSet, indexerResultSet)

	var unspentOutputs []UTXOOutput

	// Runs the next query against the indexer.
	// Returns false if there are no more results to collect.
	// Loop only adds Basic Outputs to the lists ingoingOutputs and ingoingOutputIds
	for indexerResultSet.Next() {
		Component.LogInfof("indexerResultSet: %v, %T", indexerResultSet, indexerResultSet)

		indexerResponse := indexerResultSet.Response
		Component.LogInfof("indexerResponse: %v, %T", indexerResponse, indexerResponse)
		Component.LogInfof("indexerResponse Length: %v, %T", len(indexerResponse.Items), len(indexerResponse.Items))

		outputs, err := indexerResultSet.Outputs()
		if err != nil {
			Component.LogErrorf("Failed to fetch outputs: %s", err)
		}
		Component.LogInfof("outputs: %v, %T", outputs, outputs)

		for i, output := range outputs {
			Component.LogInfof("Index: %v, %T", i, i)
			Component.LogInfof("Current OutputId: %v", indexerResponse.Items[i])

			// Only use basic outputs as inputs to a new transaction if they don't use the metadata feature
			switch o := output.(type) {
			case *iotago.BasicOutput:

				// Only use basic outputs as inputs to a new transaction if they don't use the metadata feature
				// This shall prevent that previous notarization outputs are consumed involuntarily
				if o.FeatureSet().MetadataFeature() == nil {
					Component.LogInfof("Metadata Feature? false")

					ingoingHexOutputId := iotago.HexOutputIDs{indexerResponse.Items[i]}
					Component.LogInfof("ingoingHexOutputId: %v, %T", ingoingHexOutputId, ingoingHexOutputId)

					ingoingOutputIds, err := ingoingHexOutputId.OutputIDs()
					if err != nil {
						panic(err)
					}
					Component.LogInfof("ingoingOutputIds: %v, %T", ingoingOutputIds, ingoingOutputIds)

					unspentOutput := UTXOOutput{
						OutputID: ingoingOutputIds[0],
						Output:   o,
					}
					Component.LogInfof("unspentOutput: %v, %T", unspentOutput, unspentOutput)

					unspentOutputs = append(unspentOutputs, unspentOutput)
				} else {
					Component.LogInfof("Metadata Feature? true")
				}

			}
		}

		Component.LogInfof("unspentOutputs: %v, %T", unspentOutputs, unspentOutputs)
	}

	return unspentOutputs
}

func prepTxPayload(protoParas *iotago.ProtocolParameters, unspentOutputs []UTXOOutput, address *iotago.Ed25519Address, signer iotago.AddressSigner) *iotago.Transaction {
	// Prepare Basic Output Transaction
	networkID := protoParas.NetworkID()
	Component.LogInfof("NetworkID: %v, %T", networkID, networkID)

	txBuilder := builder.NewTransactionBuilder(networkID)
	Component.LogInfof("txBuilder: %v, %T", txBuilder, txBuilder)

	var totalDeposit uint64 = 0
	for _, unspentOutput := range unspentOutputs {
		txBuilder.AddInput(&builder.TxInput{UnlockTarget: address, InputID: unspentOutput.OutputID, Input: unspentOutput.Output})
		Component.LogInfof("Added unspent output to Tx: %v, %T", unspentOutput.OutputID, unspentOutput.OutputID)

		totalDeposit += unspentOutput.Output.Deposit()
	}
	Component.LogInfof("totalDeposit: %v, %T", totalDeposit, totalDeposit)

	// Add Basic Output with Notarization Metadata
	txBuilder.AddOutput(&iotago.BasicOutput{
		Amount: uint64(1000000),
		Conditions: iotago.UnlockConditions{
			&iotago.AddressUnlockCondition{Address: address},
		},
		Features: iotago.Features{
			&iotago.MetadataFeature{Data: []byte("This is some hash commitment")},
		},
	})

	// Add Basic Output for token remainder
	txBuilder.AddOutput(&iotago.BasicOutput{
		Amount: uint64(totalDeposit - 1000000),
		Conditions: iotago.UnlockConditions{
			&iotago.AddressUnlockCondition{Address: address},
		},
	})
	Component.LogInfof("txBuilder: %v, %T", txBuilder, txBuilder)

	txPayload, err := txBuilder.Build(protoParas, signer)
	if err != nil {
		panic(err)
	}
	Component.LogInfof("txPayload: %v, %T", txPayload, txPayload)

	return txPayload
}

func prepAndSendBlock(c echo.Context, protoParas *iotago.ProtocolParameters, txPayload *iotago.Transaction) string {
	ctx := c.Request().Context()

	transactionID, err := txPayload.ID()
	if err != nil {
		panic(err)
	}
	Component.LogInfof("transactionID: %v, %T", transactionID.ToHex(), transactionID.ToHex())

	inxNodeClient := deps.NodeBridge.INXNodeClient()

	tipsResponse, err := inxNodeClient.Tips(ctx)
	if err != nil {
		panic(err)
	}

	tips, err := tipsResponse.Tips()
	if err != nil {
		panic(err)
	}

	block, err := builder.NewBlockBuilder().
		ProtocolVersion(protoParas.Version).
		Parents(tips).
		Payload(txPayload).
		Build()

	if err != nil {
		panic(err)
	}

	Component.LogInfof("block: %v, %T", block, block)

	blockId, err := inxNodeClient.SubmitBlock(ctx, block, protoParas)
	if err != nil {
		panic(err)
	}

	return blockId.ToHex()
}
