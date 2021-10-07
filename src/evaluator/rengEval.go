package evaluator

import (
	sdl "RenG/src/SDL"
	"RenG/src/ast"
	"RenG/src/object"
	"fmt"
	"sync"
)

var (
	LayerMutex = &sync.RWMutex{}
)

func RengEval(node ast.Node, root string, env *object.Environment, renderer *sdl.SDL_Renderer, LayerList *sdl.LayerList, TextureList *object.TextureList) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalRengBlockStatements(node, root, env, renderer, LayerList, TextureList)
	case *ast.ExpressionStatement:
		return RengEval(node.Expression, root, env, renderer, LayerList, TextureList)
	case *ast.ShowExpression:
		return evalShowExpression(node, root, env, renderer, LayerList, TextureList)
	}
	return nil
}

func evalRengBlockStatements(block *ast.BlockStatement, root string, env *object.Environment, renderer *sdl.SDL_Renderer, LayerList *sdl.LayerList, TextureList *object.TextureList) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = RengEval(statement, root, env, renderer, LayerList, TextureList)
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalShowExpression(node *ast.ShowExpression, apsolutedRoot string, env *object.Environment, renderer *sdl.SDL_Renderer, layerList *sdl.LayerList, textureList *object.TextureList) object.Object {
	if texture, ok := textureList.Get(node.Name.String()); ok {

		LayerMutex.Lock()
		layerList.Layers[0].AddNewTexture(texture)
		LayerMutex.Unlock()

		return nil
	}

	if root, ok := env.Get(node.Name.String()); ok {
		texture, suc := renderer.LoadFromFile(apsolutedRoot + root.(*object.Image).Root.Inspect())
		fmt.Println(apsolutedRoot + root.(*object.Image).Root.Inspect())

		if !suc {
			return nil
		}

		LayerMutex.Lock()
		layerList.Layers[0].AddNewTexture(texture)
		LayerMutex.Unlock()

		textureList.Set(node.Name.String(), texture)
	}

	return nil
}
