bucket               = "terraform-state-reciapp-globanthack"
key                  = "reciapp.tfstate"
region               = "us-east-1"
profile              = "globanthack"
dynamodb_table       = "tf_lock_state"
encrypt              = true
workspace_key_prefix = "v1"
