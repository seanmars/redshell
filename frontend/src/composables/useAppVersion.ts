import { ref } from 'vue';
import { GetAppVersion } from '@wailsjs/go/app/SystemApp';

const version = ref('');
let loaded = false;

export function useAppVersion() {
  if (!loaded) {
    loaded = true;
    GetAppVersion()
      .then((v) => {
        version.value = v;
      })
      .catch(() => {
        loaded = false;
      });
  }
  return { version };
}
