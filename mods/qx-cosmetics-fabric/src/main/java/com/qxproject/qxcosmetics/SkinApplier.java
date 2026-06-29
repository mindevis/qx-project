package com.qxproject.qxcosmetics;

import net.minecraft.client.MinecraftClient;
import net.minecraft.client.network.AbstractClientPlayerEntity;
import net.minecraft.client.util.SkinTextures;
import net.minecraft.util.Identifier;

/**
 * Applies a registered local skin texture to the current player.
 */
public final class SkinApplier {
    private SkinApplier() {}

    public static void applyLocalSkin(MinecraftClient client, Identifier skinId, boolean slim) {
        if (!(client.player instanceof AbstractClientPlayerEntity player)) {
            return;
        }
        SkinTextures.Type model = slim ? SkinTextures.Type.SLIM : SkinTextures.Type.WIDE;
        SkinTextures textures = new SkinTextures(skinId, null, null, null, model, false);
        player.setSkinTextures(textures);
    }
}
