# The engine is called through gomobile's JNI bindings: keep them whole.
-keep class go.** { *; }
-keep class com.namanvashistha.calmchess.engine.** { *; }
