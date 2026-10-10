package engine

// Hooks whose overlays are still being decoded. Each is named after the
// routine it stands in for; bodies are filled in as the decodes land.

func (e *Engine) clear67F4() { e.loot = purse{} } // DS:67F4
