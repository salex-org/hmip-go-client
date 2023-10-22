# GO Client for the Homematic IP Cloud
A Golang wrapper for the REST API of the Homematic IP Cloud.
Since there is no official documentation I used the code of the [Python wrapper](https://github.com/coreGreenberet/homematicip-rest-api)
to get an idea of how the API works. Thanks to [coreGreenberet](https://github.com/coreGreenberet) for doing the great job of
reverse engineering. **Use this library at your own risk!**

# Installation

Run the following command to install the library in your GO module:

```shell
go get github.com/salex-org/hmip-go-client
```

# Loading the configuration and getting the current state
First set the following environment variables for the HmIP-Client to connect to your Account:
| Name | Description |
|------|-------------|
| | |
| HMIP_AP_SGTIN | The SGTIN of your Access Point |
| HMIP_PIN | The PIN of your Access Point (optional, only needed when a PIN was set during setup of your device) |
| HMIP_CLIENT_ID | The client ID (will be generated when [registering a new client](#registering-a-new-client)) | 
| HMIP_CLIENT_NAME | The name of the client (will be set when [registering a new client](#registering-a-new-client)) |
| HMIP_DEVICE_ID | The device ID (will be generated when [registering a new client](#registering-a-new-client)) |
| HMIP_CLIENT_AUTH_TOKEN | The client auth token (will be generated when [registering a new client](#registering-a-new-client)) | |
| HMIP_AUTH_TOKEN | The auth token (will be generated when [registering a new client](#registering-a-new-client)) |

**You should not store any tokens or other secrets as plain text in the environment!**

As an example, you can use [age](https://github.com/FiloSottile/age) for encryption and
[sops](https://github.com/mozilla/sops) to edit the encrypted configuration.
At runtime, you can use [sops exec-env](https://github.com/mozilla/sops#passing-secrets-to-other-processes)
to decrypt the configuration on the fly and pass it as environment variables only to your process.

With the environment set you can run the following command to get the current state:
```shell
go run cmd/state/main.go
```

As an alternative, you can compile the tool and run it directly.

# Registering a new client

To register a new client you can run the following command:
```shell
go run cmd/registration/main.go
```

As an alternative, you can compile the tool and run it directly.

# Examples
Please have a look at the [code of the command line tools](/cmd) to get some examples for using the library in your code.

# Work in Progress
The development of the library is 'work in progress', so actually only a few features are available. 
