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
%sql
CREATE TABLE IF NOT EXISTS ${table_name};
```
#### Block 3
```
%python
from pyspark.sql.functions import lit
import pandas as pd
from io import StringIO

df = spark.table(table_name)

csv_data = StringIO(csv_table)
pandas_df = pd.read_csv(csv_data)
pandas_df.columns = pandas_df.columns.str.strip()
for col in pandas_df.columns:
    if pandas_df[col].dtype == 'object':
        try:
            pandas_df[col] = pd.to_datetime(pandas_df[col])
        except ValueError:
            pass
newRow = spark.createDataFrame(pandas_df)

if len(df.columns) > 0:
    append = newRow.exceptAll(df)
    append.write.saveAsTable(name = table_name, mode = 'append', mergeSchema = True)
else:
    newRow.write.saveAsTable(name = table_name, mode = 'overwrite', overwriteSchema = True)
```

## databaseConfigRef CRD
See "krateoplatformops/finops-operator-exporter".
