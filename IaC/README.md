# Terraform Reciapp

Before to launch a resource you need to have in mind  the following:

  - Install terraform, use Terraform v0.12.13, you can download it here 
     https://releases.hashicorp.com/terraform/0.12.13/
  - Configure your Terraform token to use organizations and store the state in Terraform Enterprise https://instride.atlassian.net/wiki/x/-QAaAw
  - Create a S3 bucket to store the terraform plan, if you don't want to store the plans in AWS please cahnge the backend file.


These script can be used to deploy the resources for reciapp, the resources created by scripts are:

- DynamoDBTables

## Usage

You need to clone the repository and select/create the correct Terraform Workspace which you want to create the resources. 

**1.** Clone the repository

**2.** Run Terraform init. Specify the backends configuration that you want to use.
```terraform
terraform init -backend-prod=./File_To_Use
```
**3.** Select the workspace where you want to deploy.

```terraform
terraform workspace select Workspace_Name
```

**4.** Run the terraform plan command
```terraform
terraform plan -var aws_profile=="aws_profile(according to profiles name in ~/.aws/credentials)"  -var -var Region="Region_used_to_deploy" 
```

**5.** Verify the plan 

**6.** Run terraform apply
 ```terraform
terraform apply -var aws_profile=="aws_profile(according to profiles name in ~/.aws/credentials)"  -var -var Region="Region_used_to_deploy" 
```