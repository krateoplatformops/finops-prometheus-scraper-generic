# operator-exporter
This repository is part of a wider exporting architecture for the FinOps Cost and Usage Specification (FOCUS). This component is tasked with scraping a given Prometheus endpoint. The configuration is obtained from a file mounted inside the container in "/config/config.yaml". The scraper uploads all the data to a Databricks database, as reported in the database-config field.

## Dependencies
There is the need to have an active Databricks cluster, with SQL warehouse and notebooks configured. Its login details must be placed in the database-config CR. 
The Databricks cluster also needs to have an active notebook with the code specified below.

## Configuration
To start the scraping process, see the "config-sample.yaml" file.

This container is automatically started by the operator-scraper.

## Installation
To build the executable: 
```
make build REPO=<your-registry-here>
```

To build and push the Docker images:
```
make container REPO=<your-registry-here>
```

## Databricks notebook
The field 'notebookPath' of the YAML config file needs to contain the path to a Databricks notebook with the following code in three sequential blocks:

#### Block 1
```
%python
table_name = dbutils.widgets.get('table_name')
dbutils.widgets.text("table_name", table_name)
file_name = dbutils.widgets.get('file_name')
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
```
#### Block 2
```
%python
query = ("CREATE TABLE IF NOT EXISTS " + table_name + "(",
"AvailabilityZone STRING,",
"BilledCost DOUBLE,",
"BillingAccountId STRING,",
"BillingAccountName STRING,",
"BillingCurrency STRING,",
"BillingPeriodEnd TIMESTAMP,",
"BillingPeriodStart TIMESTAMP,",
"ChargeCategory STRING,",
"ChargeDescription STRING,",
"ChargeFrequency STRING,",
"ChargePeriodEnd TIMESTAMP,",
"ChargePeriodStart TIMESTAMP,",
"ChargeSubcategory STRING,",
"CommitmentDiscountCategory STRING,",
"CommitmentDiscountId STRING,",
"CommitmentDiscountName STRING,",
"CommitmentDiscountType STRING,",
"EffectiveCost DOUBLE,",
"InvoiceIssuer STRING,",
"ListCost DOUBLE,",
"ListUnitPrice DOUBLE,",
"PricingCategory STRING,",
"PricingQuantity DOUBLE,",
"PricingUnit STRING,",
"Provider STRING,",
"Publisher STRING,",
"Region STRING,",
"ResourceId STRING,",
"ResourceName STRING,",
"ResourceType STRING,",
"ServiceCategory STRING,",
"ServiceName STRING,",
"SkuId STRING,",
"SkuPriceId STRING,",
"SubAccountId STRING,",
"SubAccountName STRING,",
"Tags STRING,",
"UsageQuantity DOUBLE,",
"UsageUnit STRING, ",
"Value DOUBLE",
");")
query = ''.join(query)
print(query)
if len(column_splits) == 40:
    spark.sql(query)
else:
    spark.sql("CREATE TABLE IF NOT EXISTS " + table_name + ";")
```
#### Block 3
```
%python
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
