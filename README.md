# INX Notarizer Plugin

The INX Notarizer Plugin is a custom extension for IOTA Hornet Nodes using the IOTA Node Extension (INX) interface. It provides functionalities related to the notarization of data on the IOTA UTXO Ledger, allowing users to notarize and verify hashes of documents or any arbitrary data, ensuring their integrity and timestamp without the need for a centralized authority.

## Features

- **Health Check**: A simple endpoint to check the health and responsiveness of the plugin.
- **Notarization**: Create a transaction including a hash of your document or data, effectively notarizing it in a Basic Output of the IOTA UTXO Ledger.
- **Verification**: Verify the notarization of a document or data by checking its hash against the stored Basic Output.

## Endpoints

The plugin exposes three main RESTful endpoints:

### 1. Health Check

- **Endpoint**: `/api/inx-notarizer/v1/health`
- **Method**: `GET`
- **Description**: Checks the health of the plugin. Useful for monitoring and operational purposes.
- **Response**: HTTP 200 OK if the plugin is up and running.

### 2. Create Notarization

- **Endpoint**: `/api/inx-notarizer/v1/create/:hash`
- **Method**: `POST`
- **Description**: Notarizes a hash by creating a transaction that creates a Basic Output which includes the hash in its metadata field. The hash should be passed as a parameter in the URL.
- **URL Parameter**: `hash` - The hash of the document or data to be notarized.
- **Response**: JSON object containing the `blockId` of the created transaction.

```json
{
  "blockId": "0x123abc..."
}
```

### 3. Verify Notarization
- **Endpoint**: `/api/inx-notarizer/v1/verify`
- **Method**: `GET`
- **Description**: Verifies the notarization of a hash by searching the Basic Output that includes the hash in its metadata field.
- **Query Parameters**:
  - `hash`: The hash of the document or data to verify.
  - `outputID`: The output ID of the Basic Output where the hash was notarized.
- **Response**: JSON object indicating whether the hash matches (true or false).

```json
{
  "match": true
}
```

##  Installation
1) Ensure you have a running IOTA node with INX enabled.
2) Clone the plugin repository to your local machine.
3) Follow the setup instructions in the repository to integrate the plugin with your IOTA node.
4) Start your IOTA node with the plugin enabled.

## Configuration
The plugin can be configured through environment variables or a configuration file. Please refer to the `.env_sample` file for an example of the available configuration options.

## Contributing
Contributions to the INX Notarizer Plugin are welcome. Please submit your pull requests to the repository or open an issue if you encounter any problems.

## License
The INX Notarizer Plugin is released under the MIT License. Please see the LICENSE file for more details.

Feel free to adjust the contents according to the actual functionalities, configuration options, and setup instructions of your INX Notarizer Plugin.
