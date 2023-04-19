// Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "Kubefirst",
            "email": "help@kubefirst.io"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/aws/domain/validate/:domain": {
            "get": {
                "description": "Returns status of whether or not an AWS hosted zone is validated for use with Kubefirst",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "aws"
                ],
                "summary": "Returns status of whether or not an AWS hosted zone is validated for use with Kubefirst",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Domain name, no trailing dot",
                        "name": "domain",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/types.AWSDomainValidateResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/types.JSONFailureResponse"
                        }
                    }
                }
            }
        },
        "/aws/profiles": {
            "get": {
                "description": "Returns a list of configured AWS profiles",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "aws"
                ],
                "summary": "Returns a list of configured AWS profiles",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/types.AWSProfilesResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/types.JSONFailureResponse"
                        }
                    }
                }
            }
        },
        "/civo/domain/validate/:domain": {
            "get": {
                "description": "Returns status of whether or not a Civo hosted zone is validated for use with Kubefirst",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "civo"
                ],
                "summary": "Returns status of whether or not a Civo hosted zone is validated for use with Kubefirst",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Domain name, no trailing dot",
                        "name": "domain",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Domain validation request in JSON format",
                        "name": "settings",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/types.CivoDomainValidationRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/types.CivoDomainValidationResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/types.JSONFailureResponse"
                        }
                    }
                }
            }
        },
        "/cluster": {
            "get": {
                "description": "Return all known configured Kubefirst clusters",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "cluster"
                ],
                "summary": "Return all known configured Kubefirst clusters",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/types.Cluster"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/types.JSONFailureResponse"
                        }
                    }
                }
            }
        },
        "/cluster/:cluster_name": {
            "get": {
                "description": "Return a configured Kubefirst cluster",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "cluster"
                ],
                "summary": "Return a configured Kubefirst cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster name",
                        "name": "cluster_name",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/types.Cluster"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/types.JSONFailureResponse"
                        }
                    }
                }
            },
            "post": {
                "description": "Create a Kubefirst cluster",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "cluster"
                ],
                "summary": "Create a Kubefirst cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster name",
                        "name": "cluster_name",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Cluster create request in JSON format",
                        "name": "definition",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/types.ClusterDefinition"
                        }
                    }
                ],
                "responses": {
                    "202": {
                        "description": "Accepted",
                        "schema": {
                            "$ref": "#/definitions/types.JSONSuccessResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/types.JSONFailureResponse"
                        }
                    }
                }
            },
            "delete": {
                "description": "Delete a Kubefirst cluster",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "cluster"
                ],
                "summary": "Delete a Kubefirst cluster",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Cluster name",
                        "name": "cluster_name",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "202": {
                        "description": "Accepted",
                        "schema": {
                            "$ref": "#/definitions/types.JSONSuccessResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/types.JSONFailureResponse"
                        }
                    }
                }
            }
        },
        "/health": {
            "get": {
                "description": "Return health status if the application is running.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "health"
                ],
                "summary": "Return health status if the application is running.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/types.JSONHealthResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "types.AWSDomainValidateResponse": {
            "type": "object",
            "properties": {
                "validated": {
                    "type": "boolean"
                }
            }
        },
        "types.AWSProfilesResponse": {
            "type": "object",
            "properties": {
                "profiles": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                }
            }
        },
        "types.CivoDomainValidationRequest": {
            "type": "object",
            "properties": {
                "cloud_region": {
                    "type": "string"
                }
            }
        },
        "types.CivoDomainValidationResponse": {
            "type": "object",
            "properties": {
                "validated": {
                    "type": "boolean"
                }
            }
        },
        "types.Cluster": {
            "type": "object",
            "properties": {
                "alertsEmail": {
                    "type": "string"
                },
                "argoCDAuthToken": {
                    "type": "string"
                },
                "argoCDCreateRegistryCheck": {
                    "type": "boolean"
                },
                "argoCDInitializeCheck": {
                    "type": "boolean"
                },
                "argoCDInstallCheck": {
                    "type": "boolean"
                },
                "argoCDPassword": {
                    "type": "string"
                },
                "argoCDUsername": {
                    "type": "string"
                },
                "atlantisWebhookSecret": {
                    "type": "string"
                },
                "atlantisWebhookURL": {
                    "type": "string"
                },
                "civoToken": {
                    "type": "string"
                },
                "cloudProvider": {
                    "type": "string"
                },
                "cloudRegion": {
                    "type": "string"
                },
                "cloudTerraformApplyCheck": {
                    "type": "boolean"
                },
                "cloudTerraformApplyFailedCheck": {
                    "type": "boolean"
                },
                "clusterID": {
                    "type": "string"
                },
                "clusterName": {
                    "type": "string"
                },
                "clusterSecretsCreatedCheck": {
                    "type": "boolean"
                },
                "clusterType": {
                    "type": "string"
                },
                "domainLivenessCheck": {
                    "type": "boolean"
                },
                "domainName": {
                    "type": "string"
                },
                "gitCredentialsCheck": {
                    "type": "boolean"
                },
                "gitHost": {
                    "type": "string"
                },
                "gitInitCheck": {
                    "description": "Checks",
                    "type": "boolean"
                },
                "gitOwner": {
                    "type": "string"
                },
                "gitProvider": {
                    "type": "string"
                },
                "gitTerraformApplyCheck": {
                    "type": "boolean"
                },
                "gitToken": {
                    "type": "string"
                },
                "gitUser": {
                    "type": "string"
                },
                "gitlabOwnerGroupID": {
                    "type": "integer"
                },
                "gitopsPushedCheck": {
                    "type": "boolean"
                },
                "gitopsReadyCheck": {
                    "type": "boolean"
                },
                "id": {
                    "type": "string"
                },
                "installToolsCheck": {
                    "type": "boolean"
                },
                "kbotSetupCheck": {
                    "type": "boolean"
                },
                "kubefirstTeam": {
                    "type": "string"
                },
                "postDetokenizeCheck": {
                    "type": "boolean"
                },
                "privateKey": {
                    "type": "string"
                },
                "publicKey": {
                    "type": "string"
                },
                "publicKeys": {
                    "type": "string"
                },
                "stateStoreCreateCheck": {
                    "type": "boolean"
                },
                "stateStoreCredentials": {
                    "$ref": "#/definitions/types.StateStoreCredentials"
                },
                "stateStoreCredsCheck": {
                    "type": "boolean"
                },
                "stateStoreDetails": {
                    "$ref": "#/definitions/types.StateStoreDetails"
                },
                "usersTerraformApplyCheck": {
                    "type": "boolean"
                },
                "vaultInitializedCheck": {
                    "type": "boolean"
                },
                "vaultTerraformApplyCheck": {
                    "type": "boolean"
                }
            }
        },
        "types.ClusterDefinition": {
            "type": "object",
            "required": [
                "admin_email",
                "cloud_provider",
                "cloud_region",
                "domain_name",
                "git_owner",
                "git_provider",
                "git_token",
                "type"
            ],
            "properties": {
                "admin_email": {
                    "type": "string"
                },
                "cloud_provider": {
                    "type": "string",
                    "enum": [
                        "aws",
                        "civo",
                        "digitalocean",
                        "k3d",
                        "vultr"
                    ]
                },
                "cloud_region": {
                    "type": "string"
                },
                "cluster_name": {
                    "type": "string"
                },
                "domain_name": {
                    "type": "string"
                },
                "git_owner": {
                    "type": "string"
                },
                "git_provider": {
                    "type": "string",
                    "enum": [
                        "github",
                        "gitlab"
                    ]
                },
                "git_token": {
                    "type": "string"
                },
                "type": {
                    "type": "string",
                    "enum": [
                        "mgmt",
                        "workload"
                    ]
                }
            }
        },
        "types.JSONFailureResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "err"
                }
            }
        },
        "types.JSONHealthResponse": {
            "type": "object",
            "properties": {
                "status": {
                    "type": "string",
                    "example": "healthy"
                }
            }
        },
        "types.JSONSuccessResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "success"
                }
            }
        },
        "types.StateStoreCredentials": {
            "type": "object",
            "properties": {
                "accessKeyID": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "secretAccessKey": {
                    "type": "string"
                }
            }
        },
        "types.StateStoreDetails": {
            "type": "object",
            "properties": {
                "hostname": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:port",
	BasePath:         "/api/v1",
	Schemes:          []string{},
	Title:            "Kubefirst API",
	Description:      "Kubefirst API",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
