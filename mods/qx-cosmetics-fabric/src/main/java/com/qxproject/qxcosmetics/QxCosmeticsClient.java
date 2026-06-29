package com.qxproject.qxcosmetics;

import net.fabricmc.api.ClientModInitializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class QxCosmeticsClient implements ClientModInitializer {
    public static final String MOD_ID = "qx-cosmetics";
    public static final Logger LOGGER = LoggerFactory.getLogger(MOD_ID);

    @Override
    public void onInitializeClient() {
        CosmeticsManager.init();
        LOGGER.info("QX Cosmetics client initialized");
    }
}
