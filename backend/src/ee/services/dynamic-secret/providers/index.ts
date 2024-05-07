import { AwsIamProvider } from "./aws-iam";
import { CassandraProvider } from "./cassandra";
import { DynamicSecretProviders } from "./models";
import { SqlDatabaseProvider } from "./sql-database";

export const buildDynamicSecretProviders = () => ({
  [DynamicSecretProviders.SqlDatabase]: SqlDatabaseProvider(),
  [DynamicSecretProviders.Cassandra]: CassandraProvider(),
  [DynamicSecretProviders.AwsIam]: AwsIamProvider()
});
