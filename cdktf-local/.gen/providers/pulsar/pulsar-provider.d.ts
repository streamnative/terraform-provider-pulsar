import { Construct } from 'constructs';
import * as cdktf from 'cdktf';
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
export declare class PulsarProvider extends cdktf.TerraformProvider {
    static readonly tfResourceType: string;
    /**
    * Create a new {@link https://www.terraform.io/docs/providers/pulsar pulsar} Resource
    *
    * @param scope The scope in which to define this construct
    * @param id The scoped construct ID. Must be unique amongst siblings in the same scope
    * @param options PulsarProviderConfig
    */
    constructor(scope: Construct, id: string, config: PulsarProviderConfig);
    private _apiVersion?;
    get apiVersion(): string | undefined;
    set apiVersion(value: string | undefined);
    resetApiVersion(): void;
    get apiVersionInput(): string | undefined;
    private _tlsAllowInsecureConnection?;
    get tlsAllowInsecureConnection(): boolean | cdktf.IResolvable | undefined;
    set tlsAllowInsecureConnection(value: boolean | cdktf.IResolvable | undefined);
    resetTlsAllowInsecureConnection(): void;
    get tlsAllowInsecureConnectionInput(): boolean | cdktf.IResolvable | undefined;
    private _tlsTrustCertsFilePath?;
    get tlsTrustCertsFilePath(): string | undefined;
    set tlsTrustCertsFilePath(value: string | undefined);
    resetTlsTrustCertsFilePath(): void;
    get tlsTrustCertsFilePathInput(): string | undefined;
    private _token?;
    get token(): string | undefined;
    set token(value: string | undefined);
    resetToken(): void;
    get tokenInput(): string | undefined;
    private _webServiceUrl;
    get webServiceUrl(): string;
    set webServiceUrl(value: string);
    get webServiceUrlInput(): string;
    private _alias?;
    get alias(): string | undefined;
    set alias(value: string | undefined);
    resetAlias(): void;
    get aliasInput(): string | undefined;
    protected synthesizeAttributes(): {
        [name: string]: any;
    };
}
