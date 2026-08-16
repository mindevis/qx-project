import { App } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';

/** Toast API from Ant Design App context (theme + locale). */
export function useMessage(): MessageInstance {
  return App.useApp().message;
}
