<template>
  <v-container>
    <v-layout wrap align-center row justify-space-between>
      <h1>Monitoring</h1>
      <v-flex xs12 sm4 d-flex v-if="nodes.length">
        <v-spacer></v-spacer>
        <v-select
          v-model="activeNode"
          :items="nodes"
          item-text="address"
          class="align-end"
        ></v-select>
      </v-flex>
    </v-layout>

    <v-container class="pa-0 pb-4" fluid grid-list-md>
      <v-data-iterator
        :items="currentSources"
        hide-actions
        content-tag="v-layout"
        row
        wrap
      >
        <template #header>
          <h2>Sources</h2>
        </template>
        <template #item="props">
          <v-flex
            xs12
            sm6
            md4
            lg3
          >
            <v-card>
              <v-card-title><h4>{{ props.item.id }} ({{ props.item.provider }})</h4></v-card-title>
              <v-divider></v-divider>
              <v-list dense>
                <v-list-tile>
                  <v-list-tile-content>State:</v-list-tile-content>
                  <v-list-tile-content
                    :class="{'red--text': props.item.state === 'dead' || props.item.state === 'finished', 'green--text': props.item.state !== 'dead' && props.item.state !== 'finished'}"
                    class="align-end"
                  >
                    {{ props.item.state }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Connection:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.connection }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Bytes Processed:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.bytesProcessed }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Inbound Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.inboundMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Outbound Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.outboundMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Inflight Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.inflightMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Dropped Messages:</v-list-tile-content>
                  <v-list-tile-content
                    :class="{'red--text': props.item.droppedMessages > 0, 'green--text': props.item.droppedMessages === 0}"
                    class="align-end"
                  >
                    {{ props.item.droppedMessages }}
                  </v-list-tile-content>
                </v-list-tile>
              </v-list>
            </v-card>
          </v-flex>
        </template>
      </v-data-iterator>
    </v-container>

    <v-container class="pa-0 pb-4" fluid grid-list-md>
      <v-data-iterator
        :items="currentTargets"
        hide-actions
        content-tag="v-layout"
        row
        wrap
      >
        <template #header>
          <h2>Targets</h2>
        </template>
        <template #item="props">
          <v-flex
            xs12
            sm6
            md4
            lg3
          >
            <v-card>
              <v-card-title><h4>{{ props.item.id }} ({{ props.item.provider }})</h4></v-card-title>
              <v-divider></v-divider>
              <v-list dense>
                <v-list-tile>
                  <v-list-tile-content>State:</v-list-tile-content>
                  <v-list-tile-content
                    :class="{'red--text': props.item.state === 'dead' || props.item.state === 'finished', 'green--text': props.item.state !== 'dead' && props.item.state !== 'finished'}"
                    class="align-end"
                  >
                    {{ props.item.state }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Connection:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.connection }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Bytes Processed:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.bytesProcessed }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Inbound Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.inboundMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Outbound Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.outboundMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Inflight Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.inflightMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Dropped Messages:</v-list-tile-content>
                  <v-list-tile-content
                    :class="{'red--text': props.item.droppedMessages > 0, 'green--text': props.item.droppedMessages === 0}"
                    class="align-end"
                  >
                    {{ props.item.droppedMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Resent Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.resentMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <template v-for="limiter in props.item.rateLimiters">
                  <v-divider :key="'divider_' + limiter.id"></v-divider>
                  <v-list-tile :key="'id_' + limiter.id">
                    <v-list-tile-content>ID:</v-list-tile-content>
                    <v-list-tile-content class="align-end">
                      {{ limiter.id }}
                    </v-list-tile-content>
                  </v-list-tile>
                  <v-list-tile :key="'limit_' + limiter.id">
                    <v-list-tile-content>Limit:</v-list-tile-content>
                    <v-list-tile-content class="align-end">
                      {{ limiter.limit }}/{{ limiter.interval }}
                    </v-list-tile-content>
                  </v-list-tile>
                  <v-list-tile :key="'average_' + limiter.id">
                    <v-list-tile-content>Average:</v-list-tile-content>
                    <v-list-tile-content
                      :class="{'red--text': limiter.averageBreached, 'green--text': !limiter.averageBreached}"
                      class="align-end"
                    >
                      {{ limiter.average }}/{{ limiter.interval }}
                    </v-list-tile-content>
                  </v-list-tile>
                  <v-list-tile :key="'current_' + limiter.id">
                    <v-list-tile-content>Current:</v-list-tile-content>
                    <v-list-tile-content
                      :class="{'red--text': limiter.currentBreached, 'green--text': !limiter.currentBreached}"
                      class="align-end"
                    >
                      {{ limiter.current }}/{{ limiter.interval }}
                    </v-list-tile-content>
                  </v-list-tile>
                  <v-list-tile :key="'stored_' + limiter.id">
                    <v-list-tile-content>Stored Metrics:</v-list-tile-content>
                    <v-list-tile-content class="align-end">
                      {{ limiter.storedMetrics }}x{{ limiter.interval }}
                    </v-list-tile-content>
                  </v-list-tile>
                </template>
              </v-list>
            </v-card>
          </v-flex>
        </template>
      </v-data-iterator>
    </v-container>

    <v-container class="pa-0 pb-4" fluid grid-list-md>
      <v-data-iterator
        :items="currentWorkers"
        hide-actions
        content-tag="v-layout"
        row
        wrap
      >
        <template #header>
          <h2>Workers</h2>
        </template>
        <template #item="props">
          <v-flex
            xs12
            sm6
            md4
            lg3
          >
            <v-card>
              <v-card-title><h4>{{ props.item.id }}</h4></v-card-title>
              <v-divider></v-divider>
              <v-list dense>
                <v-list-tile>
                  <v-list-tile-content>State:</v-list-tile-content>
                  <v-list-tile-content
                    :class="{'red--text': props.item.state === 'dead' || props.item.state === 'finished', 'green--text': props.item.state !== 'dead' && props.item.state !== 'finished'}"
                    class="align-end"
                  >
                    {{ props.item.state }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Bytes Processed:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.bytesProcessed }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Inbound Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.inboundMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Outbound Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.outboundMessages }}
                  </v-list-tile-content>
                </v-list-tile>
                <v-list-tile>
                  <v-list-tile-content>Inflight Messages:</v-list-tile-content>
                  <v-list-tile-content class="align-end">
                    {{ props.item.inflightMessages }}
                  </v-list-tile-content>
                </v-list-tile>
              </v-list>
            </v-card>
          </v-flex>
        </template>
      </v-data-iterator>
    </v-container>
  </v-container>
</template>

<script>
export default {
  data: () => ({
    updateInterval: null,
    sources: [],
    targets: [],
    workers: [],
    nodes: [],
    nodeData: {},
    currentNode: "",
    currentData: {},
  }),
  mounted() {
    let baseUrl = '/api';

    if (process.env.VUE_APP_API_BASE) {
      baseUrl = process.env.VUE_APP_API_BASE;
    }

    this.loadData(baseUrl);
  },
  computed: {
    activeNode: {
      set(value) {
        this.currentNode = value;
        this.currentData = this.nodeData[value];
      },
      get() {
        return this.currentNode;
      }
    },
    currentSources() {
      if (this.currentNode.length === 0) {
        return this.sources;
      }

      return this.currentData.sources;
    },
    currentTargets() {
      if (this.currentNode.length === 0) {
        return this.targets;
      }

      return this.currentData.targets;
    },
    currentWorkers() {
      if (this.currentNode.length === 0) {
        return this.workers;
      }

      return this.currentData.workers;
    },
  },
  methods: {
    loadData(baseUrl) {
      fetch(`${baseUrl}/nodes`).then(response => response.json()).then((data) => {
        if (this.nodes.length === 0) {
          this.nodes = [];

          fetch(`${baseUrl}/workers`).then(response => response.json()).then((data) => {
            this.workers = data.workers;
          });

          fetch(`${baseUrl}/sources`).then(response => response.json()).then((data) => {
            this.sources = data.sources;
          });

          fetch(`${baseUrl}/targets`).then(response => response.json()).then((data) => {
            this.targets = data.targets;
          });
        } else {
          this.nodes = data.nodes;

          data.nodes.forEach((node) => {
            if (this.currentNode.length > 0 && this.currentNode !== node.address) {
              return
            }
            if (!this.nodeData[node.address]) {
              this.nodeData[node.address] = {
                workers: [],
                targets: [],
                sources: [],
              };
            }

            let promise1 = fetch(`${baseUrl}/workers?node=${node.address}`).then(response => response.json()).then((data) => {
              this.nodeData[node.address].workers = data.workers;
            });

            let promise2 = fetch(`${baseUrl}/sources?node=${node.address}`).then(response => response.json()).then((data) => {
              this.nodeData[node.address].sources = data.sources;
            });

            let promise3 = fetch(`${baseUrl}/targets?node=${node.address}`).then(response => response.json()).then((data) => {
              this.nodeData[node.address].targets = data.targets;
            });

            Promise.all([promise1, promise2, promise3]).then(() => {
              if (this.currentNode.length === 0) {
                this.currentNode = node.address;
              }

              if (this.currentNode.length > 0 && this.currentNode === node.address) {
                this.currentData = this.nodeData[node.address];
              }
            })
          });
        }

        setTimeout(() => {
          this.loadData(baseUrl);
        }, 1000);
      });
    },
  },
};
</script>
