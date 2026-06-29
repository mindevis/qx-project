package com.qxproject.qxcosmetics;

import com.google.gson.Gson;
import com.google.gson.annotations.SerializedName;
import net.fabricmc.fabric.api.client.event.lifecycle.v1.ClientTickEvents;
import net.minecraft.client.MinecraftClient;
import net.minecraft.client.texture.NativeImage;
import net.minecraft.client.texture.NativeImageBackedTexture;
import net.minecraft.util.Identifier;

import java.io.IOException;
import java.io.Reader;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Locale;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Loads qx/{uuid}.json from the game directory and registers custom skin/cape textures.
 */
public final class CosmeticsManager {
    private static final Gson GSON = new Gson();
    private static final Map<String, PlayerCosmetics> CACHE = new ConcurrentHashMap<>();
    private static boolean loaded;

    private CosmeticsManager() {}

    public static void init() {
        ClientTickEvents.END_CLIENT_TICK.register(client -> {
            if (loaded || client.player == null) {
                return;
            }
            loaded = true;
            reload(client);
        });
    }

    public static void reload(MinecraftClient client) {
        if (client.runDirectory == null || client.player == null) {
            return;
        }
        Path qxDir = client.runDirectory.toPath().resolve("qx");
        if (!Files.isDirectory(qxDir)) {
            return;
        }
        String uuid = client.player.getUuidAsString().replace("-", "").toLowerCase(Locale.ROOT);
        Path meta = qxDir.resolve(uuid + ".json");
        if (!Files.isRegularFile(meta)) {
            return;
        }
        try (Reader reader = Files.newBufferedReader(meta)) {
            PlayerCosmetics cosmetics = GSON.fromJson(reader, PlayerCosmetics.class);
            if (cosmetics == null) {
                return;
            }
            CACHE.put(uuid, cosmetics);
            applySkin(client, cosmetics);
            QxCosmeticsClient.LOGGER.info("Loaded QX cosmetics for {}", cosmetics.username);
        } catch (IOException e) {
            QxCosmeticsClient.LOGGER.warn("Failed to read cosmetics meta {}", meta, e);
        }
    }

    private static void applySkin(MinecraftClient client, PlayerCosmetics cosmetics) {
        if (cosmetics.skinFile == null || cosmetics.skinFile.isBlank()) {
            return;
        }
        Path skinPath = Path.of(cosmetics.skinFile);
        if (!Files.isRegularFile(skinPath)) {
            return;
        }
        try (NativeImage image = NativeImage.read(Files.newInputStream(skinPath))) {
            Identifier id = Identifier.of(QxCosmeticsClient.MOD_ID, "skin/local");
            NativeImageBackedTexture texture = new NativeImageBackedTexture(image);
            client.getTextureManager().registerTexture(id, texture);
            SkinApplier.applyLocalSkin(client, id, "alex".equalsIgnoreCase(cosmetics.model));
        } catch (IOException e) {
            QxCosmeticsClient.LOGGER.warn("Failed to load skin {}", skinPath, e);
        }
    }

    public static PlayerCosmetics getForPlayer(String uuidNoDash) {
        return CACHE.get(uuidNoDash.toLowerCase(Locale.ROOT));
    }

    public static final class PlayerCosmetics {
        public String uuid;
        public String username;
        public String model;
        @SerializedName("skin_file")
        public String skinFile;
        @SerializedName("skin_url")
        public String skinUrl;
        public String cape;
        @SerializedName("cape_file")
        public String capeFile;
        public String wings;
    }
}
