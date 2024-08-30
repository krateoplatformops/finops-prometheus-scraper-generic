# FinOps Prometheus Scraper Generic
This repository is part of the wider exporting architecture for the Krateo Composable FinOps and scrapes Prometheus exporters to then upload the data to a data lake.

## Summary
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Configuration](#configuration)

## Overview
This component is tasked with scraping a given Prometheus endpoint. The configuration is obtained from a file mounted inside the container in "/config/config.yaml". The scraper uploads all the data to a Databricks database, as reported in the database-config field.

## Architecture
![Krateo Composable FinOps Prometheus Scraper Generic](resources/images/KCF-scraper.png)

## Configuration
This container is automatically started by the operator-scraper.

To build the executable: 
```
make build REPO=<your-registry-here>
```

To build and push the Docker images:
```
make container REPO=<your-registry-here>
```

### Dependencies
There is the need to have an active Databricks cluster, with SQL warehouse and notebooks configured. Its login details must be placed in the database-config CR. 
The Databricks cluster also needs to have an active notebook with the code specified below.

### Databricks notebook
The field 'notebookPath' of the YAML config file needs to contain the path to a Databricks notebook with the following code in three sequential blocks:

```python
table_name = dbutils.widgets.get('table_name')
dbutils.widgets.text("table_name", table_name)
file_name = dbutils.widgets.get('file_name')
import os 
with open('/dbfs'+file_name, 'r') as table_file:
    table = table_file.read()
rows = str.split(table, '\n')
i = 0
header = ''
output = ''
for row in rows:
    values_tuple = ()
    column_splits = str.split(row, ',')
    for column_split in column_splits:
        keyValueSplit = str.split(column_split, '=')
        if keyValueSplit[0] != 'Value': # Drop Prometheus formatting artifact
            if i == 0:
                header += keyValueSplit[0] + ','
            if len(keyValueSplit) == 2:
                output += keyValueSplit[1] + ','
            else:
                output += ','
    i+=1
    output = output[:-1]
    output += '\n'
header = header[:-1]
csv_table = header + '\n' + output
query = ("CREATE TABLE IF NOT EXISTS " + table_name + "(",
"AvailabilityZone STRING,",
"BilledCost DOUBLE,",
"BillingAccountId STRING,",
"BillingAccountName STRING,",
"BillingCurrency STRING,",
"BillingPeriodEnd TIMESTAMP,",
"BillingPeriodStart TIMESTAMP,",
"ChargeCategory STRING,",
"ChargeClass STRING,",
"ChargeDescription STRING,",
"ChargeFrequency STRING,",
"ChargePeriodEnd TIMESTAMP,",
"ChargePeriodStart TIMESTAMP,",
"CommitmentDiscountCategory STRING,",
"CommitmentDiscountId STRING,",
"CommitmentDiscountName STRING,",
"CommitmentDiscountStatus STRING,",
"CommitmentDiscountType STRING,",
"ConsumedUnit STRING,",
"ConsumedQuantity STRING,",
"ContractedCost DOUBLE,",
"ContractedUnitCost DOUBLE,",
"EffectiveCost DOUBLE,",
"InvoiceIssuerName STRING,",
"ListCost DOUBLE,",
"ListUnitPrice DOUBLE,",
"PricingCategory STRING,",
"PricingQuantity DOUBLE,",
"PricingUnit STRING,",
"ProviderName STRING,",
"PublisherName STRING,",
"RegionId STRING,",
"RegionName STRING,",
"ResourceId STRING,",
"ResourceName STRING,",
"ResourceType STRING,",
"ServiceCategory STRING,",
"ServiceName STRING,",
"SkuId STRING,",
"SkuPriceId STRING,",
"SubAccountId STRING,",
"SubAccountName STRING,",
"Tags STRING",
");")
query = ''.join(query)
if len(column_splits) == 43:
    spark.sql(query)
else:
    spark.sql("CREATE TABLE IF NOT EXISTS " + table_name + ";")
from pyspark.sql.functions import lit, col
import pandas as pd
from io import StringIO
df = spark.table(table_name)
csv_data = StringIO(csv_table)
pandas_df = pd.read_csv(csv_data)
pandas_df.columns = pandas_df.columns.str.strip()
for col_name in pandas_df.columns:
    if pandas_df[col_name].dtype == 'object':
        try:
            pandas_df[col_name] = pd.to_datetime(pandas_df[col_name])
        except ValueError:
            pass
newRow = spark.createDataFrame(pandas_df)
if len(df.columns) > 0:
    append = newRow.exceptAll(df)
    append.write.saveAsTable(name = table_name, mode = 'append', mergeSchema = True)
else:
    newRow.write.saveAsTable(name = table_name, mode = 'overwrite', overwriteSchema = True)
```