package notarizer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/adrian-grassl/inx-notarizer/pkg/hdwallet"
	"github.com/iotaledger/hive.go/logger"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/builder"
	"github.com/iotaledger/iota.go/v3/nodeclient"
	"github.com/labstack/echo/v4"
)

// Global variable for the plugin's logger
var Logger *logger.Logger

// Initialize the plugin's logger
func init() {
	cfg := logger.DefaultCfg

	globalLogger, err := logger.NewRootLogger(cfg)
	if err != nil {
		panic(err)
	}

	logger.SetGlobalLogger(globalLogger)

	Logger = logger.NewLogger("Notarizer")
}

type UTXOOutput struct {
	OutputID iotago.OutputID
	Output   iotago.Output
}

type BasicOutput struct {
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

func getHealth(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func createNotarization(c echo.Context) error {
	// Extract the hash parameter from the request path
	hash := c.Param("hash")
	Logger.Infof("Received hash for notarization: %s", hash)

	protoParas := deps.NodeBridge.ProtocolParameters()
	Logger.Infof("protoParas: %v, %T", protoParas, protoParas)

	// Load mnemonic from .env
	mnemonic, err := loadEnvVariable("MNEMONIC")
	if err != nil {
		Logger.Errorf("Error loading mnemonic: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error loading mnemonic")
	}
	Logger.Debug("Mnemonic loaded successfully")

	// Prepare wallet address and signer
	walletObject, err := prepWallet(protoParas, mnemonic)
	if err != nil {
		Logger.Errorf("Error preparing wallet: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error preparing wallet")
	}

	// Fetch outputs for address
	indexerResultSet, err := fetchOutputsByAddress(walletObject.Bech32Address)
	if err != nil {
		Logger.Errorf("Error fetching outputs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error fetching outputs")
	}

	// Filter outputs for their eligibility to become input to the tx.
	unspentOutputs, err := filterOutputs(indexerResultSet)
	if err != nil {
		Logger.Errorf("Error filtering outputs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error filtering outputs")
	}

	// Prepare transaction payload including the notarization hash.
	txPayload, err := prepTxPayload(protoParas, unspentOutputs, walletObject.Ed25519Address, walletObject.AddressSigner, hash)
	if err != nil {
		Logger.Errorf("Error preparing transaction payload: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error preparing transaction payload")
	}

	// Prepare and send the block with the notarization transaction.
	hexBlockId, err := prepAndSendBlock(c, protoParas, txPayload)
	if err != nil {
		Logger.Errorf("Error sending block: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error sending block")
	}
	Logger.Infof("Block attached with ID: %v", hexBlockId)

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
func prepWallet(protoParas *iotago.ProtocolParameters, mnemonic []string) (*WalletObject, error) {

	wallet, err := hdwallet.NewHDWallet(protoParas, mnemonic, "", 0, false)
	if err != nil {
		return nil, fmt.Errorf("creating wallet failed, err: %s", err)
	}
	Logger.Debugf("Wallet created successfully")

	address, signer, err := wallet.Ed25519AddressAndSigner(0)
	if err != nil {
		return nil, fmt.Errorf("deriving ed25519 address and signer failed, err: %s", err)
	}
	Logger.Debugf("Address and signer derived successfully")

	bech32 := address.Bech32("tst")
	Logger.Debugf("Bech32 Address: %v, %T", bech32, bech32)

	return &WalletObject{
		Bech32Address:  bech32,
		Ed25519Address: address,
		AddressSigner:  signer,
	}, nil
}

// fetchOutputsByAddress fetches the unspent outputs associated with a certain address.
func fetchOutputsByAddress(bech32 string) ([]UTXOOutput, error) {
	ctxIndexer, cancelIndexer := context.WithTimeout(context.Background(), indexerPluginAvailableTimeout)
	defer cancelIndexer()

	ctxRequest, cancelRequest := context.WithTimeout(context.Background(), inxRequestTimeout)
	defer cancelRequest()

	basicOutputsQuery := &nodeclient.BasicOutputsQuery{
		AddressBech32: bech32,
	}
	Logger.Infof("Fetching UTXO outputs for address: %s", bech32)

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
			ingoingHexOutputId := iotago.HexOutputIDs{indexerResultSet.Response.Items[i]}
			ingoingOutputIds, err := ingoingHexOutputId.OutputIDs()
			if err != nil {
				panic(err)
			}

			unspentOutputs = append(unspentOutputs, UTXOOutput{
				OutputID: ingoingOutputIds[0],
				Output:   output,
			})
		}
	}

	if indexerResultSet.Error != nil {
		return nil, fmt.Errorf("indexer result set error: %v", indexerResultSet.Error)
	}

	return unspentOutputs, nil
}

// filterOutputs filters the list of unspent outputs that can be used as input to a new tx.
func filterOutputs(unspentOutputs []UTXOOutput) ([]BasicOutput, error) {
	var suitableOutputs []BasicOutput
	for _, unspentOutput := range unspentOutputs {
		if basicOutput, ok := unspentOutput.Output.(*iotago.BasicOutput); ok && basicOutput.FeatureSet().MetadataFeature() == nil {
			suitableOutputs = append(suitableOutputs, BasicOutput{
				OutputID: unspentOutput.OutputID,
				Output:   basicOutput,
			})
		}
	}

	return suitableOutputs, nil
}

// prepTxPayload prepares the transaction payload by incorporating the notarization hash and creating outputs.
func prepTxPayload(protoParas *iotago.ProtocolParameters, unspentOutputs []BasicOutput, address *iotago.Ed25519Address, signer iotago.AddressSigner, hash string) (*iotago.Transaction, error) {
	txBuilder := builder.NewTransactionBuilder(protoParas.NetworkID())
	Logger.Infof("Building transaction with network ID: %v", protoParas.NetworkID)

	var totalDeposit uint64 = 0
	for _, unspentOutput := range unspentOutputs {
		txBuilder.AddInput(&builder.TxInput{
			UnlockTarget: address,
			InputID:      unspentOutput.OutputID,
			Input:        unspentOutput.Output,
		})
		totalDeposit += unspentOutput.Output.Deposit()
	}

	// Prepare Basic Output that will hold notarization hash in its metadata and add it to the transaction
	notarizationOutput := &iotago.BasicOutput{
		Conditions: iotago.UnlockConditions{
			&iotago.AddressUnlockCondition{Address: address},
		},
		Features: iotago.Features{
			&iotago.MetadataFeature{Data: []byte(hash)},
		},
	}

	notarizationOutputCost := protoParas.RentStructure.MinRent(notarizationOutput)
	Logger.Infof("notarizationOutputCost: %v", notarizationOutputCost)

	notarizationOutput.Amount = notarizationOutputCost

	// Add a basic output with the notarization hash as metadata.
	txBuilder.AddOutput(notarizationOutput)

	// Add a basic output that holds the token deposit remainder.
	if remainder := totalDeposit - notarizationOutputCost; remainder > 0 {
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
	Logger.Infof("Transaction ID: %s", transactionID.ToHex())

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
