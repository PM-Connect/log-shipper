<template>
  <v-app dark>
    <v-toolbar app>
      <v-toolbar-title class="headline text-uppercase">
        <span>LOG</span>
        <span class="font-weight-light">SHIPPER</span>
      </v-toolbar-title>
      <v-spacer></v-spacer>
      <span class="mr-2">Latest Release</span>
    </v-toolbar>

    <v-content>
      <router-view/>
    </v-content>
  </v-app>
</template>

<script>
export default {
  name: 'App',
  data() {
    return {
      //
    };
  },
};
</script>
