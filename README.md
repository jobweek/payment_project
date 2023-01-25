# Payment Project

**A small service MVP, which allows the user to deposit(tx_type="deposit"), withdraw funds(tx_type="withdraw"), get information about transactions by `transaction id` and the current balance of the user by `user id`.**

### Run in Docker compose

```
# docker-compose up
```

### Workflow

#### Step 1 - Registration User and Deposit

POST : `/v1.0/payment`

BODY : `{
"user_id":string,
"amount":number,
"tx_type":string
}`

EXAMPLE:  

http://localhost:8088/v1.0/payment

{
"user_id":"Ilyha",
"amount":100,
"tx_type":"deposit"
}

#### Step 2 - Get User

GET : `/v1.0/user?id=user_id`

EXAMPLE:  

http://localhost:8088/v1.0/user?id=Ilyha

#### Step 3 - Get Transaction Info by id

GET : `/v1.0/payment?id=tx_id`

EXAMPLE:

http://localhost:8088/v1.0/payment?id=5bf8c5ce-b9c7-4c64-949a-e9655e82bcfd

#### Step 4 - Withdraw

POST : `/v1.0/payment`

BODY : `{
"user_id":string,
"amount":number,
"tx_type":string
}`

EXAMPLE:

http://localhost:8088/v1.0/payment

{
"user_id":"Ilyha",
"amount":50,
"tx_type":"withdraw"
}

### Pjoject Architecture Visualisation

![Image](documentation_resources/v1.0.png)