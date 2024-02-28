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
columns_list = []
values_list = []

for row in rows:
    values_tuple = ()
    column_splits = str.split(row, ',')
    for column_split in column_splits:
        keyValueSplit = str.split(column_split, '=')
        if len(keyValueSplit) == 2:
            values_tuple = (*values_tuple, keyValueSplit[1])
        else:
            values_tuple = (*values_tuple, '')
        if keyValueSplit[0] not in columns_list:
            columns_list.append(keyValueSplit[0])
    values_list.append(values_tuple)
```
#### Block 2
```
%sql
CREATE TABLE IF NOT EXISTS ${table_name};
```
#### Block 3
```
%python
from pyspark.sql.functions import lit
df = spark.table(table_name)

if len(df.columns) == 0:
    for column_key in columns_list:
        df = df.withColumn(column_key, lit(''))

newRow = spark.createDataFrame(values_list, columns_list)
append = newRow.exceptAll(df)
append.write.saveAsTable(name = table_name, mode = 'append', mergeSchema = True)
```

## databaseConfigRef CRD
See "krateoplatformops/finops-operator-exporter".
