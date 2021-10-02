import { Construct } from "constructs";
import { App, TerraformStack } from "cdktf";
import {PulsarProvider, Tenant} from "./.gen/providers/pulsar"

class MyStack extends TerraformStack {
  constructor(scope: Construct, name: string) {
    super(scope, name);

    new PulsarProvider(this, 'pulsar', {
      webServiceUrl: 'http://localhost:8080'
    })

    // define resources here
    new Tenant(this, 'my-tenant', {
      tenant: 'my-tenant',
      allowedClusters: ["standalone"],
    })
  }
}

const app = new App();
new MyStack(app, "cdktf-local");
app.synth();
