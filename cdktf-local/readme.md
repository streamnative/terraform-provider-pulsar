Using CDK with Pulsar-Terraform-Provider<br />
Prerequisites:

1. Terraform >= v0.12
2. Node.js >= v12.16
3. Yarn >= v1.21

Steps:<br/>
1. Install cdktf
```
brew install cdktf
```
2. Initialize cdktf project
```
mkdir cdktf-pulsar
cd cdktf-pulsar
cdktf init --template typescript --local
```
3. Define Pulsar Terraform Provider<br/>
Open file _cdktf.json_ and update terraformProviders to be
```
  "terraformProviders": ["quantummetric/pulsar@~>1.0.0"],
```
Then run code generation
```
cdktf get
```
4. Define infrastructures, now cdktf has generate code for our terraform provider _pulsar_ in our selected language _typescript_, we can add our infrastructure in _main.ts_:
Add import:
```
import {PulsarProvider, Tenant} from "./.gen/providers/pulsar"
```
Add resources:
```
new PulsarProvider(this, 'pulsar', {
      webServiceUrl: 'http://localhost:8080'
    })

    // define resources here
    new Tenant(this, 'my-tenant', {
      tenant: 'my-tenant',
      allowedClusters: ["standalone"],
    })
```
5. Execute, run command
```
cdktf deploy
```
the tool will translate our code into terraform and execute it.

cdktf only provide 2 options to store state:Terraform Cloud and local, but there're some instruction on how to migrate local state to a remote backend: https://github.com/hashicorp/terraform-cdk/blob/048d176b93aae909383bf85dc6e1d4ad57bb4077/docs/working-with-cdk-for-terraform/remote-backend.md