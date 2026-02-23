<div align="center">

<div style="
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 25px;
  padding: 50px 30px;
  margin: 30px auto;
  max-width: 600px;
  box-shadow: 0 25px 50px rgba(0,0,0,0.15);
  position: relative;
  overflow: hidden;
">
  <div style="
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: url('data:image/svg+xml,<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 100 100\"><defs><pattern id=\"grain\" width=\"100\" height=\"100\" patternUnits=\"userSpaceOnUse\"><circle cx=\"20\" cy=\"20\" r=\"2\" fill=\"%23ffffff\" opacity=\"0.1\"/><circle cx=\"80\" cy=\"80\" r=\"1.5\" fill=\"%23ffffff\" opacity=\"0.08\"/><circle cx=\"40\" cy=\"60\" r=\"1\" fill=\"%23ffffff\" opacity=\"0.06\"/></pattern></defs><rect width=\"100\" height=\"100\" fill=\"url(%23grain)\"/></svg>');
    opacity: 0.3;
  "></div>
  
  <div style="
    background: rgba(255,255,255,0.95);
    border-radius: 20px;
    padding: 40px 30px;
    position: relative;
    z-index: 2;
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255,255,255,0.3);
  ">
    <div style="
      background: linear-gradient(45deg, #e8f5e8 0%, #a8e6cf 100%);
      border-radius: 50%;
      padding: 25px;
      display: inline-block;
      margin-bottom: 25px;
      box-shadow: 0 15px 35px rgba(168,230,207,0.3);
      border: 3px solid rgba(168,230,207,0.4);
      position: relative;
    ">
      <img src="../.assets/Friendly Ailloy with Glowing Orb.png" alt="Ailloy Documentation" width="150" style="
        display: block;
        border-radius: 50%;
        transition: transform 0.4s cubic-bezier(0.68, -0.55, 0.265, 1.55);
      " onmouseover="this.style.transform='scale(1.1) rotate(5deg)'" onmouseout="this.style.transform='scale(1) rotate(0deg)'"/>
      
      <div style="
        position: absolute;
        top: -5px;
        right: -5px;
        background: linear-gradient(45deg, #4CAF50, #8BC34A);
        color: white;
        border-radius: 50%;
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 20px;
        font-weight: bold;
        box-shadow: 0 8px 20px rgba(76,175,80,0.4);
        animation: pulse 2s infinite;
      ">📚</div>
    </div>
    
    <h1 style="
      font-size: 2.8em;
      margin: 15px 0;
      background: linear-gradient(135deg, #667eea, #764ba2);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
      text-shadow: 2px 2px 4px rgba(0,0,0,0.1);
    ">Ailloy Documentation</h1>
    
    <div style="
      background: linear-gradient(90deg, #667eea, #764ba2);
      height: 3px;
      width: 80px;
      margin: 20px auto;
      border-radius: 2px;
    "></div>
    
    <p style="
      color: #666;
      font-size: 1.1em;
      margin-top: 20px;
      line-height: 1.6;
    ">Comprehensive guides for the package manager for AI instructions</p>
  </div>
</div>

<style>
@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}
</style>

</div>

These guides teach you how to create, package, and share your own AI workflow packages with Ailloy. For a quick overview of the project, see the [main README](../README.md).

## Getting Started

- [Blanks](blanks.md) — What blanks are and how to create commands, skills, and workflows

## Authoring Guides

- [Flux Variables](flux.md) — Configure blanks with variables, schemas, and value layering
- [Ingots](ingots.md) — Create and use reusable template components
- [Packaging Molds](smelt.md) — Package molds into distributable tarballs or binaries

## Operations

- [Remote Molds](foundry.md) — Resolve molds from git repositories with semver constraints
- [Configuration Wizard](anneal.md) — Interactive wizard for flux variable configuration
- [Validation](temper.md) — Lint and validate mold and ingot packages
- [Plugins](plugin.md) — Generate Claude Code plugins from molds