provider "aws" {
  profile = "${var.aws_profile}"
  region  = "${var.region}"
}

####### DynamoDB  #####

####### User table  #####
resource "aws_dynamodb_table" "user-dynamodb-table" {
  name           = "users"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 2
  hash_key       = "id"
  
  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "username"
    type = "S"
  }

  global_secondary_index {
    name               = "by_username"
    hash_key           = "username"
    write_capacity     = 2
    read_capacity      = 2
    projection_type    = "ALL"
  }

  tags = {
    Name        = "env"
    Environment = "recyapp"
  }
}

####### Location table  #####

resource "aws_dynamodb_table" "location-dynamodb-table" {
  name           = "locations"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 2
  hash_key       = "id"
  
  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    Name        = "env"
    Environment = "recyapp"
  }
}

####### recycling package table  #####

resource "aws_dynamodb_table" "recyclingpackage-dynamodb-table" {
  name           = "recycling_packages"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 2
  hash_key       = "id"
  
  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    Name        = "env"
    Environment = "recyapp"
  }
}


####### UserLocation table  #####

resource "aws_dynamodb_table" "UserLocation-dynamodb-table" {
  name           = "user_locations"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 2
  hash_key       = "id"
  
  attribute {
    name = "id"
    type = "S"
  }

  global_secondary_index {
    name               = "by_userid"
    hash_key           = "id"
    write_capacity     = 2
    read_capacity      = 2
    projection_type    = "ALL"
  }  

  tags = {
    Name        = "env"
    Environment = "recyapp"
  }
}


####### PickingRoute table  #####

resource "aws_dynamodb_table" "PickingRoute-dynamodb-table" {
  name           = "picking_routes"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 2
  hash_key       = "id"
  
  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    Name        = "env"
    Environment = "recyapp"
  }
}

####### LocationNeedsPickup table  #####
resource "aws_dynamodb_table" "LocationNeedsPickup-dynamodb-table" {
  name           = "location_needs_pickup"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 2
  hash_key       = "id"
  
  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    Name        = "env"
    Environment = "recyapp"
  }
}


####### LocationbalanceMovements table  #####
resource "aws_dynamodb_table" "LocationbalanceMovements-dynamodb-table" {
  name           = "location_balance_movements"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 2
  hash_key       = "id"
  
  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    Name        = "env"
    Environment = "recyapp"
  }
}