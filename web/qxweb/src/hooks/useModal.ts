import { App } from 'antd';

/** Modal.confirm / info / error from Ant Design App context (theme + locale). */
export function useModal() {
  return App.useApp().modal;
}
