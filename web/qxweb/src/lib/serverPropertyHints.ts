import { DEFAULT_LOCALE, type Locale } from '@/i18n';

export type ServerPropertyMeta = {
  title: string;
  description: string;
};

const metaEn: Record<string, ServerPropertyMeta> = {
  'accepts-transfers': {
    title: 'Accept transfers',
    description: 'Allow players to join this server via a transfer packet from another server.',
  },
  'allow-flight': {
    title: 'Allow flight',
    description: 'Lets Survival players fly (needed for some mods). Without it, flying players are kicked after a few seconds.',
  },
  'allow-nether': {
    title: 'Allow Nether',
    description: 'Whether players can travel to the Nether through portals.',
  },
  'announce-player-achievements': {
    title: 'Announce advancements',
    description: 'Broadcast player advancements and achievements in chat.',
  },
  'broadcast-console-to-ops': {
    title: 'Broadcast console to ops',
    description: 'Send console command output to all online operators.',
  },
  'broadcast-rcon-to-ops': {
    title: 'Broadcast RCON to ops',
    description: 'Send RCON command output to all online operators.',
  },
  'bug-report-link': {
    title: 'Bug report link',
    description: 'URL shown to players for reporting bugs. Leave empty to hide the link.',
  },
  'chat-spam-threshold-seconds': {
    title: 'Chat spam threshold',
    description: 'How quickly chat spam is punished. 0 disables automatic kicks for chat spam.',
  },
  'command-spam-threshold-seconds': {
    title: 'Command spam threshold',
    description: 'How quickly command spam is punished. 0 disables automatic kicks for command spam.',
  },
  debug: {
    title: 'Debug mode',
    description: 'Enables extra debug output in the server log. Leave off unless you are diagnosing a problem.',
  },
  difficulty: {
    title: 'Difficulty',
    description: 'World difficulty: peaceful, easy, normal, or hard. Affects mob damage, hunger, and poison.',
  },
  'enable-code-of-conduct': {
    title: 'Code of conduct',
    description: 'Show a code-of-conduct screen from the server’s codeofconduct folder before players join.',
  },
  'enable-command-block': {
    title: 'Command blocks',
    description: 'Allow command blocks to run commands on the server.',
  },
  'enable-jmx-monitoring': {
    title: 'JMX monitoring',
    description: 'Expose tick-time metrics over JMX for external monitoring tools.',
  },
  'enable-query': {
    title: 'Enable query',
    description: 'Turn on the GameSpy query protocol so server lists and tools can read status.',
  },
  'enable-rcon': {
    title: 'Enable RCON',
    description: 'Allow remote console access over the network. Do not expose RCON to the public internet.',
  },
  'enable-status': {
    title: 'Show in server list',
    description: 'If off, the server looks offline in the multiplayer list but still accepts connections.',
  },
  'enforce-secure-profile': {
    title: 'Secure chat profiles',
    description: 'Only allow players with a Mojang-signed chat key. Required for reportable signed chat.',
  },
  'enforce-whitelist': {
    title: 'Enforce whitelist',
    description: 'Kick players who are not on the whitelist as soon as the list is reloaded.',
  },
  'entity-broadcast-range-percentage': {
    title: 'Entity tracking range',
    description: 'How far entities are sent to players, as a percent of the default (10–1000). Higher values use more bandwidth.',
  },
  'force-gamemode': {
    title: 'Force default gamemode',
    description: 'Always put joining players into the default gamemode instead of the mode they left in.',
  },
  'function-permission-level': {
    title: 'Function permission level',
    description: 'Permission level used by datapack functions (1–4).',
  },
  gamemode: {
    title: 'Default gamemode',
    description: 'Mode for new players: survival, creative, adventure, or spectator.',
  },
  'generate-structures': {
    title: 'Generate structures',
    description: 'Generate villages, strongholds, and other structures in new chunks. Dungeons can still appear if this is off.',
  },
  'generator-settings': {
    title: 'Generator settings',
    description: 'JSON used to customize world generation, mainly for flat or single-biome worlds.',
  },
  hardcore: {
    title: 'Hardcore',
    description: 'Players are banned on death and difficulty is locked to hard.',
  },
  'hide-online-players': {
    title: 'Hide online players',
    description: 'Do not send the player list in server-list status responses.',
  },
  'initial-disabled-packs': {
    title: 'Disabled datapacks',
    description: 'Comma-separated datapacks that should stay off when a new world is created.',
  },
  'initial-enabled-packs': {
    title: 'Enabled datapacks',
    description: 'Comma-separated datapacks enabled on world creation. Feature packs must be listed explicitly.',
  },
  'level-name': {
    title: 'World folder',
    description: 'Folder name of the world save. Changing this loads or creates another world.',
  },
  'level-seed': {
    title: 'World seed',
    description: 'Seed used when generating a new world. Leave empty for a random seed. Does not change an existing world.',
  },
  'level-type': {
    title: 'World type',
    description: 'World preset: normal, flat, large_biomes, amplified, or single_biome_surface.',
  },
  'log-ips': {
    title: 'Log IP addresses',
    description: 'Write connecting players’ IP addresses to the console and log file.',
  },
  'management-server-allowed-origins': {
    title: 'Management CORS origins',
    description: 'Allowed browser origins for the Minecraft Server Management Protocol.',
  },
  'management-server-enabled': {
    title: 'Management protocol',
    description: 'Enable the Minecraft Server Management Protocol API.',
  },
  'management-server-host': {
    title: 'Management host',
    description: 'Host address the management protocol listens on.',
  },
  'management-server-port': {
    title: 'Management port',
    description: 'Port for the management protocol. 0 means the server chooses one.',
  },
  'management-server-secret': {
    title: 'Management secret',
    description: 'Shared secret for management-protocol clients. Generated automatically if left empty.',
  },
  'management-server-tls-enabled': {
    title: 'Management TLS',
    description: 'Encrypt the management protocol with TLS.',
  },
  'management-server-tls-keystore': {
    title: 'Management TLS keystore',
    description: 'Path to the TLS keystore file. Required if management TLS is enabled.',
  },
  'management-server-tls-keystore-password': {
    title: 'Management TLS password',
    description: 'Password for the management TLS keystore.',
  },
  'max-chained-neighbor-updates': {
    title: 'Neighbor update limit',
    description: 'How many chained block updates can run in a row before extras are skipped. Negative values remove the limit.',
  },
  'max-players': {
    title: 'Max players',
    description: 'Maximum number of players online at once.',
  },
  'max-tick-time': {
    title: 'Watchdog tick limit',
    description: 'Milliseconds a single tick may take before the watchdog stops the server. -1 disables the watchdog.',
  },
  'max-world-size': {
    title: 'Max world size',
    description: 'World-border radius in blocks from the center (up to 29 999 984).',
  },
  motd: {
    title: 'Message of the day',
    description: 'Text shown under the server name in the multiplayer list. Supports color codes; keep it reasonably short.',
  },
  'network-compression-threshold': {
    title: 'Network compression',
    description: 'Packets larger than this many bytes are compressed. -1 disables compression, 0 compresses everything.',
  },
  'online-mode': {
    title: 'Online mode',
    description: 'Only Mojang-authenticated (premium) accounts can join. Turn off only for offline/cracked setups.',
  },
  'op-permission-level': {
    title: 'Operator permission level',
    description: 'Permission level given to operators (1–4). 4 is full server access.',
  },
  'pause-when-empty-seconds': {
    title: 'Pause when empty',
    description: 'Seconds after the last player leaves before the server pauses. -1 or 0 keeps it running.',
  },
  'player-idle-timeout': {
    title: 'Idle kick timeout',
    description: 'Minutes of inactivity before a player is kicked. 0 disables idle kicks.',
  },
  'prevent-proxy-connections': {
    title: 'Block proxy connections',
    description: 'Reject players who appear to connect through a proxy or VPN.',
  },
  pvp: {
    title: 'PvP',
    description: 'Whether players can damage each other.',
  },
  'query.port': {
    title: 'Query port',
    description: 'UDP port for query responses. Usually the same as the game port.',
  },
  'rate-limit': {
    title: 'Packet rate limit',
    description: 'Maximum packets per second a client may send. 0 disables the limit.',
  },
  'rcon.password': {
    title: 'RCON password',
    description: 'Password required to use the remote console. Use a strong unique password.',
  },
  'rcon.port': {
    title: 'RCON port',
    description: 'TCP port for RCON connections (default 25575).',
  },
  'region-file-compression': {
    title: 'Region compression',
    description: 'Compression used for region files, for example deflate or lz4.',
  },
  'require-resource-pack': {
    title: 'Require resource pack',
    description: 'Players must accept the server resource pack or they cannot join.',
  },
  'resource-pack': {
    title: 'Resource pack URL',
    description: 'Direct download URL of the resource pack offered to joining players.',
  },
  'resource-pack-id': {
    title: 'Resource pack ID',
    description: 'Optional UUID that identifies this resource pack to clients.',
  },
  'resource-pack-prompt': {
    title: 'Resource pack prompt',
    description: 'Message shown when the client asks the player to accept the resource pack.',
  },
  'resource-pack-sha1': {
    title: 'Resource pack SHA-1',
    description: 'SHA-1 hash of the pack file so clients can verify the download.',
  },
  'server-ip': {
    title: 'Bind address',
    description: 'Network interface to listen on. Leave empty to accept connections on all addresses.',
  },
  'server-port': {
    title: 'Server port',
    description: 'TCP port the game listens on (default 25565). Restart the server after changing it.',
  },
  'simulation-distance': {
    title: 'Simulation distance',
    description: 'How many chunks around players stay loaded and tick (typically 3–32). Higher values use more CPU.',
  },
  'snooper-enabled': {
    title: 'Send usage data',
    description: 'Older setting that sent anonymous usage data to Mojang. Unused on modern versions.',
  },
  'spawn-animals': {
    title: 'Spawn animals',
    description: 'Whether passive animals such as cows and pigs spawn in the world.',
  },
  'spawn-monsters': {
    title: 'Spawn monsters',
    description: 'Whether hostile mobs spawn in the world.',
  },
  'spawn-npcs': {
    title: 'Spawn villagers',
    description: 'Whether villagers and wandering traders spawn in the world.',
  },
  'spawn-protection': {
    title: 'Spawn protection',
    description: 'Radius around spawn where only operators can break or place blocks. 0 disables protection.',
  },
  'status-heartbeat-interval': {
    title: 'Status heartbeat',
    description: 'How often the server refreshes its status heartbeat. 0 uses the default interval.',
  },
  'sync-chunk-writes': {
    title: 'Synchronous chunk saves',
    description: 'Save chunk data synchronously. Safer if the process crashes, but a bit slower.',
  },
  'text-filtering-config': {
    title: 'Chat filter config',
    description: 'Path or settings for the optional chat text filter.',
  },
  'text-filtering-version': {
    title: 'Chat filter version',
    description: 'Version of the chat filtering protocol to use.',
  },
  'use-native-transport': {
    title: 'Native network I/O',
    description: 'Use optimized Linux native networking. Leave on unless you have a compatibility issue.',
  },
  'view-distance': {
    title: 'View distance',
    description: 'How many chunks around each player are sent to the client (2–32). Higher values use more bandwidth and RAM.',
  },
  'white-list': {
    title: 'Whitelist',
    description: 'Only players on the whitelist can join. Operators can still join.',
  },
};

const metaRu: Record<string, ServerPropertyMeta> = {
  'accepts-transfers': {
    title: 'Принимать переносы',
    description: 'Разрешает игрокам заходить на этот сервер по пакету переноса с другого сервера.',
  },
  'allow-flight': {
    title: 'Разрешить полёт',
    description: 'Позволяет летать в режиме выживания (нужно для части модов). Иначе летящего игрока кикнет через несколько секунд.',
  },
  'allow-nether': {
    title: 'Нижний мир',
    description: 'Можно ли переходить в Нижний мир через порталы.',
  },
  'announce-player-achievements': {
    title: 'Объявлять достижения',
    description: 'Писать в чат, когда игрок получает достижение.',
  },
  'broadcast-console-to-ops': {
    title: 'Консоль операторам',
    description: 'Дублировать вывод команд консоли всем операторам онлайн.',
  },
  'broadcast-rcon-to-ops': {
    title: 'RCON операторам',
    description: 'Дублировать вывод команд RCON всем операторам онлайн.',
  },
  'bug-report-link': {
    title: 'Ссылка для багов',
    description: 'Адрес, по которому игроки могут сообщить об ошибке. Пустое поле скрывает ссылку.',
  },
  'chat-spam-threshold-seconds': {
    title: 'Антиспам чата',
    description: 'Насколько быстро наказывается спам в чате. 0 — игроков за спам не кикают.',
  },
  'command-spam-threshold-seconds': {
    title: 'Антиспам команд',
    description: 'Насколько быстро наказывается спам командами. 0 — кик за спам команд выключен.',
  },
  debug: {
    title: 'Режим отладки',
    description: 'Пишет дополнительную отладочную информацию в лог. Включайте только при диагностике.',
  },
  difficulty: {
    title: 'Сложность',
    description: 'Сложность мира: peaceful, easy, normal или hard. Влияет на урон мобов, голод и яд.',
  },
  'enable-code-of-conduct': {
    title: 'Кодекс поведения',
    description: 'Показывать правила из папки codeofconduct перед входом на сервер.',
  },
  'enable-command-block': {
    title: 'Командные блоки',
    description: 'Разрешает командным блокам выполнять команды на сервере.',
  },
  'enable-jmx-monitoring': {
    title: 'Мониторинг JMX',
    description: 'Открывает метрики времени тика через JMX для внешних систем мониторинга.',
  },
  'enable-query': {
    title: 'Протокол query',
    description: 'Включает протокол GameSpy query, чтобы списки серверов и утилиты могли читать статус.',
  },
  'enable-rcon': {
    title: 'Удалённая консоль (RCON)',
    description: 'Разрешает доступ к консоли по сети. Не открывайте RCON в интернет без защиты.',
  },
  'enable-status': {
    title: 'Показывать в списке',
    description: 'Если выключить, в списке серверов сервер выглядит офлайн, но подключения всё равно принимаются.',
  },
  'enforce-secure-profile': {
    title: 'Подписанный чат',
    description: 'Пускать только игроков с ключом чата от Mojang. Нужно для жалоб на сообщения.',
  },
  'enforce-whitelist': {
    title: 'Строгий белый список',
    description: 'Сразу кикать игроков, которых нет в белом списке, после его перезагрузки.',
  },
  'entity-broadcast-range-percentage': {
    title: 'Дальность сущностей',
    description: 'На каком расстоянии сущности отправляются игроку, в процентах от стандарта (10–1000). Больше — выше нагрузка.',
  },
  'force-gamemode': {
    title: 'Принудительный режим',
    description: 'При входе всегда ставить игроку режим по умолчанию, а не тот, в котором он вышел.',
  },
  'function-permission-level': {
    title: 'Права функций',
    description: 'Уровень прав для функций датапаков (1–4).',
  },
  gamemode: {
    title: 'Режим игры',
    description: 'Режим для новых игроков: survival, creative, adventure или spectator.',
  },
  'generate-structures': {
    title: 'Генерация структур',
    description: 'Создавать деревни, крепости и другие структуры в новых чанках. Подземелья могут появиться и без этого.',
  },
  'generator-settings': {
    title: 'Настройки генератора',
    description: 'JSON для тонкой настройки мира — обычно для плоского или однобиомного мира.',
  },
  hardcore: {
    title: 'Хардкор',
    description: 'После смерти игрок банится, сложность фиксируется на hard.',
  },
  'hide-online-players': {
    title: 'Скрыть список игроков',
    description: 'Не отдавать список игроков в ответе для списка серверов.',
  },
  'initial-disabled-packs': {
    title: 'Отключённые датапаки',
    description: 'Датапаки через запятую, которые не включать при создании нового мира.',
  },
  'initial-enabled-packs': {
    title: 'Включённые датапаки',
    description: 'Датапаки через запятую, которые включить при создании мира. Feature-паки нужно указывать явно.',
  },
  'level-name': {
    title: 'Папка мира',
    description: 'Имя папки сохранения. Если сменить, сервер загрузит или создаст другой мир.',
  },
  'level-seed': {
    title: 'Сид мира',
    description: 'Сид для генерации нового мира. Пустое поле — случайный сид. Уже созданный мир не меняет.',
  },
  'level-type': {
    title: 'Тип мира',
    description: 'Пресет генерации: normal, flat, large_biomes, amplified или single_biome_surface.',
  },
  'log-ips': {
    title: 'Писать IP в лог',
    description: 'Записывать IP подключающихся игроков в консоль и лог-файл.',
  },
  'management-server-allowed-origins': {
    title: 'CORS управления',
    description: 'Разрешённые источники браузера для протокола управления сервером.',
  },
  'management-server-enabled': {
    title: 'Протокол управления',
    description: 'Включает API протокола управления Minecraft-сервером.',
  },
  'management-server-host': {
    title: 'Хост управления',
    description: 'Адрес, на котором слушает протокол управления.',
  },
  'management-server-port': {
    title: 'Порт управления',
    description: 'Порт протокола управления. 0 — выбрать автоматически.',
  },
  'management-server-secret': {
    title: 'Секрет управления',
    description: 'Общий секрет для клиентов протокола управления. Если пусто, создаётся автоматически.',
  },
  'management-server-tls-enabled': {
    title: 'TLS управления',
    description: 'Шифровать протокол управления с помощью TLS.',
  },
  'management-server-tls-keystore': {
    title: 'Keystore TLS',
    description: 'Путь к файлу keystore для TLS. Нужен, если TLS управления включён.',
  },
  'management-server-tls-keystore-password': {
    title: 'Пароль keystore',
    description: 'Пароль от keystore для TLS протокола управления.',
  },
  'max-chained-neighbor-updates': {
    title: 'Лимит соседних обновлений',
    description: 'Сколько цепочек обновления блоков можно выполнить подряд. Отрицательное значение снимает лимит.',
  },
  'max-players': {
    title: 'Максимум игроков',
    description: 'Сколько игроков могут быть на сервере одновременно.',
  },
  'max-tick-time': {
    title: 'Лимит тика (watchdog)',
    description: 'Сколько миллисекунд может длиться один тик, прежде чем сервер будет остановлен. -1 отключает watchdog.',
  },
  'max-world-size': {
    title: 'Размер мира',
    description: 'Радиус границы мира в блоках от центра (до 29 999 984).',
  },
  motd: {
    title: 'Сообщение дня',
    description: 'Текст под названием сервера в списке серверов. Поддерживает цветовые коды; лучше не делать слишком длинным.',
  },
  'network-compression-threshold': {
    title: 'Сжатие сети',
    description: 'Пакеты больше этого размера в байтах сжимаются. -1 выключает сжатие, 0 сжимает всё.',
  },
  'online-mode': {
    title: 'Лицензионные аккаунты',
    description: 'Пускать только аккаунты, проверенные Mojang. Выключайте только для офлайн / пиратских сборок.',
  },
  'op-permission-level': {
    title: 'Уровень операторов',
    description: 'Уровень прав операторов (1–4). 4 — полный доступ к серверу.',
  },
  'pause-when-empty-seconds': {
    title: 'Пауза без игроков',
    description: 'Через сколько секунд после выхода последнего игрока сервер ставится на паузу. -1 или 0 — не останавливать.',
  },
  'player-idle-timeout': {
    title: 'Кик за бездействие',
    description: 'Сколько минут бездействия до кика игрока. 0 — не кикать за простой.',
  },
  'prevent-proxy-connections': {
    title: 'Блокировать прокси',
    description: 'Отклонять игроков, которые заходят через прокси или VPN.',
  },
  pvp: {
    title: 'PvP',
    description: 'Могут ли игроки наносить урон друг другу.',
  },
  'query.port': {
    title: 'Порт query',
    description: 'UDP-порт для ответов query. Обычно совпадает с игровым портом.',
  },
  'rate-limit': {
    title: 'Лимит пакетов',
    description: 'Максимум пакетов в секунду от одного клиента. 0 — без ограничения.',
  },
  'rcon.password': {
    title: 'Пароль RCON',
    description: 'Пароль для удалённой консоли. Используйте сложный уникальный пароль.',
  },
  'rcon.port': {
    title: 'Порт RCON',
    description: 'TCP-порт для подключений RCON (по умолчанию 25575).',
  },
  'region-file-compression': {
    title: 'Сжатие региона',
    description: 'Алгоритм сжатия файлов региона, например deflate или lz4.',
  },
  'require-resource-pack': {
    title: 'Обязательный ресурспак',
    description: 'Без принятия серверного ресурспака игрок не сможет зайти.',
  },
  'resource-pack': {
    title: 'URL ресурспака',
    description: 'Прямая ссылка на ресурспак, который предлагается игрокам при входе.',
  },
  'resource-pack-id': {
    title: 'ID ресурспака',
    description: 'Необязательный UUID, по которому клиент узнаёт этот ресурспак.',
  },
  'resource-pack-prompt': {
    title: 'Текст ресурспака',
    description: 'Сообщение, которое клиент показывает при запросе принять ресурспак.',
  },
  'resource-pack-sha1': {
    title: 'SHA-1 ресурспака',
    description: 'Контрольная сумма SHA-1 файла пака, чтобы клиент проверил загрузку.',
  },
  'server-ip': {
    title: 'Адрес привязки',
    description: 'Сетевой интерфейс, на котором слушает сервер. Пустое поле — принимать подключения на всех адресах.',
  },
  'server-port': {
    title: 'Порт сервера',
    description: 'TCP-порт игры (по умолчанию 25565). После смены перезапустите сервер.',
  },
  'simulation-distance': {
    title: 'Дистанция симуляции',
    description: 'Сколько чанков вокруг игрока остаются загруженными и обрабатываются (обычно 3–32). Больше — выше нагрузка на CPU.',
  },
  'snooper-enabled': {
    title: 'Отправка статистики',
    description: 'Старая настройка отправки анонимной статистики Mojang. В новых версиях не используется.',
  },
  'spawn-animals': {
    title: 'Спавн животных',
    description: 'Появляются ли в мире мирные животные — коровы, свиньи и другие.',
  },
  'spawn-monsters': {
    title: 'Спавн монстров',
    description: 'Появляются ли в мире враждебные мобы.',
  },
  'spawn-npcs': {
    title: 'Спавн жителей',
    description: 'Появляются ли в мире жители и странствующие торговцы.',
  },
  'spawn-protection': {
    title: 'Защита спавна',
    description: 'Радиус вокруг точки возрождения, где ломать и ставить блоки могут только операторы. 0 — без защиты.',
  },
  'status-heartbeat-interval': {
    title: 'Интервал статуса',
    description: 'Как часто сервер обновляет свой статус. 0 — стандартный интервал.',
  },
  'sync-chunk-writes': {
    title: 'Синхронное сохранение чанков',
    description: 'Сохранять чанки сразу на диск. Надёжнее при сбое, но чуть медленнее.',
  },
  'text-filtering-config': {
    title: 'Фильтр чата',
    description: 'Путь или настройки необязательного фильтра текста в чате.',
  },
  'text-filtering-version': {
    title: 'Версия фильтра чата',
    description: 'Версия протокола фильтрации чата.',
  },
  'use-native-transport': {
    title: 'Нативная сеть',
    description: 'Использовать ускоренный сетевой стек Linux. Оставляйте включённым, если нет проблем совместимости.',
  },
  'view-distance': {
    title: 'Дальность прорисовки',
    description: 'Сколько чанков вокруг игрока отправляется клиенту (2–32). Больше — выше расход сети и памяти.',
  },
  'white-list': {
    title: 'Белый список',
    description: 'Заходить могут только игроки из белого списка. Операторы проходят в любом случае.',
  },
};

const metaByLocale: Record<Locale, Record<string, ServerPropertyMeta>> = {
  en: metaEn,
  ru: metaRu,
};

export function getServerPropertyMeta(
  locale: Locale,
  key: string,
): { title: string; description?: string } {
  const meta = metaByLocale[locale]?.[key] ?? metaByLocale[DEFAULT_LOCALE]?.[key];
  if (!meta) {
    return { title: key };
  }
  return meta;
}

export function getServerPropertyHint(locale: Locale, key: string): string | undefined {
  return getServerPropertyMeta(locale, key).description;
}
