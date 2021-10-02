// https://www.terraform.io/docs/providers/pulsar
// generated from terraform resource schema

import { Construct } from 'constructs';
import * as cdktf from 'cdktf';

// Configuration

export interface PulsarProviderConfig {
  /**
  * Api Version to be used for the pulsar admin interaction
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar#api_version PulsarProvider#api_version}
  */
  readonly apiVersion?: string;
  /**
  * Boolean flag to accept untrusted TLS certificates
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar#tls_allow_insecure_connection PulsarProvider#tls_allow_insecure_connection}
  */
  readonly tlsAllowInsecureConnection?: boolean | cdktf.IResolvable;
  /**
  * Path to a custom trusted TLS certificate file
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar#tls_trust_certs_file_path PulsarProvider#tls_trust_certs_file_path}
  */
  readonly tlsTrustCertsFilePath?: string;
  /**
  * Authentication Token used to grant terraform permissions
to modify Apace Pulsar Entities
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar#token PulsarProvider#token}
  */
  readonly token?: string;
  /**
  * Web service url is used to connect to your apache pulsar cluster
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar#web_service_url PulsarProvider#web_service_url}
  */
  readonly webServiceUrl: string;
  /**
  * Alias name
  * 
  * Docs at Terraform Registry: {@link https://www.terraform.io/docs/providers/pulsar#alias PulsarProvider#alias}
  */
  readonly alias?: string;
}

/**
* Represents a {@link https://www.terraform.io/docs/providers/pulsar pulsar}
*/
export class PulsarProvider extends cdktf.TerraformProvider {

  // =================
  // STATIC PROPERTIES
  // =================
  public static readonly tfResourceType: string = "pulsar";

  // ===========
  // INITIALIZER
  // ===========

  /**
  * Create a new {@link https://www.terraform.io/docs/providers/pulsar pulsar} Resource
  *
  * @param scope The scope in which to define this construct
  * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
  * @param options PulsarProviderConfig
  */
  public constructor(scope: Construct, id: string, config: PulsarProviderConfig) {
    super(scope, id, {
      terraformResourceType: 'pulsar',
      terraformGeneratorMetadata: {
        providerName: 'pulsar',
        providerVersionConstraint: '~>1.0.0'
      },
      terraformProviderSource: 'quantummetric/pulsar'
    });
    this._apiVersion = config.apiVersion;
    this._tlsAllowInsecureConnection = config.tlsAllowInsecureConnection;
    this._tlsTrustCertsFilePath = config.tlsTrustCertsFilePath;
    this._token = config.token;
    this._webServiceUrl = config.webServiceUrl;
    this._alias = config.alias;
  }

  // ==========
  // ATTRIBUTES
  // ==========

  // api_version - computed: false, optional: true, required: false
  private _apiVersion?: string;
  public get apiVersion() {
    return this._apiVersion;
  }
  public set apiVersion(value: string  | undefined) {
    this._apiVersion = value;
  }
  public resetApiVersion() {
    this._apiVersion = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get apiVersionInput() {
    return this._apiVersion
  }

  // tls_allow_insecure_connection - computed: false, optional: true, required: false
  private _tlsAllowInsecureConnection?: boolean | cdktf.IResolvable;
  public get tlsAllowInsecureConnection() {
    return this._tlsAllowInsecureConnection;
  }
  public set tlsAllowInsecureConnection(value: boolean | cdktf.IResolvable  | undefined) {
    this._tlsAllowInsecureConnection = value;
  }
  public resetTlsAllowInsecureConnection() {
    this._tlsAllowInsecureConnection = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get tlsAllowInsecureConnectionInput() {
    return this._tlsAllowInsecureConnection
  }

  // tls_trust_certs_file_path - computed: false, optional: true, required: false
  private _tlsTrustCertsFilePath?: string;
  public get tlsTrustCertsFilePath() {
    return this._tlsTrustCertsFilePath;
  }
  public set tlsTrustCertsFilePath(value: string  | undefined) {
    this._tlsTrustCertsFilePath = value;
  }
  public resetTlsTrustCertsFilePath() {
    this._tlsTrustCertsFilePath = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get tlsTrustCertsFilePathInput() {
    return this._tlsTrustCertsFilePath
  }

  // token - computed: false, optional: true, required: false
  private _token?: string;
  public get token() {
    return this._token;
  }
  public set token(value: string  | undefined) {
    this._token = value;
  }
  public resetToken() {
    this._token = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get tokenInput() {
    return this._token
  }

  // web_service_url - computed: false, optional: false, required: true
  private _webServiceUrl: string;
  public get webServiceUrl() {
    return this._webServiceUrl;
  }
  public set webServiceUrl(value: string) {
    this._webServiceUrl = value;
  }
  // Temporarily expose input value. Use with caution.
  public get webServiceUrlInput() {
    return this._webServiceUrl
  }

  // alias - computed: false, optional: true, required: false
  private _alias?: string;
  public get alias() {
    return this._alias;
  }
  public set alias(value: string  | undefined) {
    this._alias = value;
  }
  public resetAlias() {
    this._alias = undefined;
  }
  // Temporarily expose input value. Use with caution.
  public get aliasInput() {
    return this._alias
  }

  // =========
  // SYNTHESIS
  // =========

  protected synthesizeAttributes(): { [name: string]: any } {
    return {
      api_version: cdktf.stringToTerraform(this._apiVersion),
      tls_allow_insecure_connection: cdktf.booleanToTerraform(this._tlsAllowInsecureConnection),
      tls_trust_certs_file_path: cdktf.stringToTerraform(this._tlsTrustCertsFilePath),
      token: cdktf.stringToTerraform(this._token),
      web_service_url: cdktf.stringToTerraform(this._webServiceUrl),
      alias: cdktf.stringToTerraform(this._alias),
    };
  }
}
