module.exports = {
  apps: [{
    name: 'eckwmsgo',
    script: './eckwmsgo',
    cwd: '/var/www/eckwmsgo',
    env: {
      NODE_ENV: 'production',
      PORT: '3210',
      JWT_SECRET: '68695b04ccf8da689a0c1fd000a05941e958bcf5d1091285400a7db43bb5402a',
      ENC_KEY: '2f8cffbfb357cb957a427fc6669d6f92100fdd471d1ed2d2',
      PG_HOST: 'localhost',
      PG_PORT: '5432',
      PG_DATABASE: 'eckwms_global',
      PG_USERNAME: 'wms_user',
      PG_PASSWORD: 'gK76543n2PqX5bV9zR4m',
      DB_ALTER: 'true',
      INSTANCE_SUFFIX: 'GO',
      GLOBAL_SERVER_URL: 'https://pda.repair'
    }
  }]
};
