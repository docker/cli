package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestMarshallConfig(t *testing.T) {
	cfg := fullExampleConfig("/foo", "/bar")
	expected := `configs: {}
networks:
  external-network:
    name: external-network
    external: true
  other-external-network:
    name: my-cool-network
    external: true
  other-network:
    driver: overlay
    driver_opts:
      baz: "1"
      foo: bar
    ipam:
      driver: overlay
      config:
      - subnet: 172.16.238.0/24
      - subnet: 2001:3984:3989::/64
  some-network: {}
secrets: {}
services:
  foo:
    build:
      context: ./dir
      dockerfile: Dockerfile
      args:
        foo: bar
      labels:
        FOO: BAR
      cache_from:
      - foo
      - bar
      network: foo
      target: foo
    cap_add:
    - ALL
    cap_drop:
    - NET_ADMIN
    - SYS_ADMIN
    cgroup_parent: m-executor-abcd
    command:
    - bundle
    - exec
    - thin
    - -p
    - "3000"
    container_name: my-web-container
    depends_on:
    - db
    - redis
    deploy:
      mode: replicated
      replicas: 6
      labels:
        FOO: BAR
      update_config:
        parallelism: 3
        delay: 10s
        failure_action: continue
        monitor: 1m0s
        max_failure_ratio: 0.3
        order: start-first
      resources:
        limits:
          cpus: "0.001"
          memory: "52428800"
        reservations:
          cpus: "0.0001"
          memory: "20971520"
          generic_resources:
          - discrete_resource_spec:
              kind: gpu
              value: 2
          - discrete_resource_spec:
              kind: ssd
              value: 1
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 2m0s
      placement:
        constraints:
        - node=foo
        preferences:
        - spread: node.labels.az
      endpoint_mode: dnsrr
    devices:
    - /dev/ttyUSB0:/dev/ttyUSB0
    dns:
    - 8.8.8.8
    - 9.9.9.9
    dns_search:
    - dc1.example.com
    - dc2.example.com
    domainname: foo.com
    entrypoint:
    - /code/entrypoint.sh
    - -p
    - "3000"
    environment:
      BAR: bar_from_env_file_2
      BAZ: baz_from_service_def
      FOO: foo_from_env_file
      QUX: qux_from_environment
    env_file:
    - ./example1.env
    - ./example2.env
    expose:
    - "3000"
    - "8000"
    external_links:
    - redis_1
    - project_db_1:mysql
    - project_db_1:postgresql
    extra_hosts:
    - somehost:162.242.195.82
    - otherhost:50.31.209.229
    hostname: foo
    healthcheck:
      test:
      - CMD-SHELL
      - echo "hello world"
      timeout: 1s
      interval: 10s
      retries: 5
      start_period: 15s
    image: redis
    ipc: host
    labels:
      com.example.description: Accounting webapp
      com.example.empty-label: ""
      com.example.number: "42"
    links:
    - db
    - db:database
    - redis
    logging:
      driver: syslog
      options:
        syslog-address: tcp://192.168.0.42:123
    mac_address: 02:42:ac:11:65:43
    network_mode: container:0cfeab0f748b9a743dc3da582046357c6ef497631c1a016d28d2bf9b4f899f7b
    networks:
      other-network:
        ipv4_address: 172.16.238.10
        ipv6_address: 2001:3984:3989::10
      other-other-network: null
      some-network:
        aliases:
        - alias1
        - alias3
    pid: host
    ports:
    - mode: ingress
      target: 3000
      protocol: tcp
    - mode: ingress
      target: 3001
      protocol: tcp
    - mode: ingress
      target: 3002
      protocol: tcp
    - mode: ingress
      target: 3003
      protocol: tcp
    - mode: ingress
      target: 3004
      protocol: tcp
    - mode: ingress
      target: 3005
      protocol: tcp
    - mode: ingress
      target: 8000
      published: 8000
      protocol: tcp
    - mode: ingress
      target: 8080
      published: 9090
      protocol: tcp
    - mode: ingress
      target: 8081
      published: 9091
      protocol: tcp
    - mode: ingress
      target: 22
      published: 49100
      protocol: tcp
    - mode: ingress
      target: 8001
      published: 8001
      protocol: tcp
    - mode: ingress
      target: 5000
      published: 5000
      protocol: tcp
    - mode: ingress
      target: 5001
      published: 5001
      protocol: tcp
    - mode: ingress
      target: 5002
      published: 5002
      protocol: tcp
    - mode: ingress
      target: 5003
      published: 5003
      protocol: tcp
    - mode: ingress
      target: 5004
      published: 5004
      protocol: tcp
    - mode: ingress
      target: 5005
      published: 5005
      protocol: tcp
    - mode: ingress
      target: 5006
      published: 5006
      protocol: tcp
    - mode: ingress
      target: 5007
      published: 5007
      protocol: tcp
    - mode: ingress
      target: 5008
      published: 5008
      protocol: tcp
    - mode: ingress
      target: 5009
      published: 5009
      protocol: tcp
    - mode: ingress
      target: 5010
      published: 5010
      protocol: tcp
    privileged: true
    read_only: true
    restart: always
    security_opt:
    - label=level:s0:c100,c200
    - label=type:svirt_apache_t
    stdin_open: true
    stop_grace_period: 20s
    stop_signal: SIGUSR1
    tmpfs:
    - /run
    - /tmp
    tty: true
    ulimits:
      nofile:
        soft: 20000
        hard: 40000
      nproc: 65535
    user: someone
    volumes:
    - type: volume
      target: /var/lib/mysql
    - type: bind
      source: /opt/data
      target: /var/lib/mysql
    - type: bind
      source: /foo
      target: /code
    - type: bind
      source: /foo/static
      target: /var/www/html
    - type: bind
      source: /bar/configs
      target: /etc/configs/
      read_only: true
    - type: volume
      source: datavolume
      target: /var/lib/mysql
    - type: bind
      source: /foo/opt
      target: /opt
      consistency: cached
    - type: tmpfs
      target: /opt
      tmpfs:
        size: 10000
    working_dir: /code
volumes:
  another-volume:
    name: user_specified_name
    driver: vsphere
    driver_opts:
      baz: "1"
      foo: bar
  external-volume:
    name: external-volume
    external: true
  external-volume3:
    name: this-is-volume3
    external: true
  other-external-volume:
    name: my-cool-volume
    external: true
  other-volume:
    driver: flocker
    driver_opts:
      baz: "1"
      foo: bar
  some-volume: {}
`

	actual, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(actual))

	// Make sure the expected still
	dict, err := ParseYAML([]byte("version: '3.6'\n" + expected))
	assert.NoError(t, err)
	_, err = Load(buildConfigDetails(dict, map[string]string{}))
	assert.NoError(t, err)
}
