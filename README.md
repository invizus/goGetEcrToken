# Simple script to get AWS ECR credentials
and store them as kubernetes secret.

This app must run in kubernetes cluster.

Currently supports only one account and region. I suggest you name your cronjob
or deployed app as "ecr-env-region-whatever" and name your secrets accordingly.

define these env variables:

- `VAR_NAMESPACE` - target namespace where to store secret, i.e. shared-secrets
- `VAR_SECRETNAME` - name of the secret, i.e. ecr-dev-us1
- `AWS_ACCESS_KEY_ID` - AWS API key ID of your env account
- `AWS_SECRET_ACCESS_KEY` - AWS API key
- `AWS_REGION` - AWS region
