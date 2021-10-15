package rengeval

import (
	sdl "RenG/src/SDL"
	"RenG/src/ast"
	"RenG/src/evaluator"
	"RenG/src/object"
	"RenG/src/transform"
	"sync"
)

var (
	LayerMutex = &sync.RWMutex{}
)

func RengEval(node ast.Node, root string, env *object.Environment, renderer *sdl.SDL_Renderer, LayerList *sdl.LayerList, TextureList *object.TextureList, Width, Height int64) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalRengBlockStatements(node, root, env, renderer, LayerList, TextureList, Width, Height)
	case *ast.ExpressionStatement:
		return RengEval(node.Expression, root, env, renderer, LayerList, TextureList, Width, Height)
	case *ast.InfixExpression:
		return evaluator.Eval(node, env)
	case *ast.PrefixExpression:
		return evaluator.Eval(node, env)
	case *ast.CallExpression:
		return evaluator.Eval(node, env)
	case *ast.ShowExpression:
		return evalShowExpression(node, root, env, renderer, LayerList, TextureList, Width, Height)
	}
	return nil
}

func evalRengBlockStatements(block *ast.BlockStatement, root string, env *object.Environment, renderer *sdl.SDL_Renderer, LayerList *sdl.LayerList, TextureList *object.TextureList, Width, Height int64) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = RengEval(statement, root, env, renderer, LayerList, TextureList, Width, Height)
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalShowExpression(node *ast.ShowExpression, apsolutedRoot string, env *object.Environment, renderer *sdl.SDL_Renderer, layerList *sdl.LayerList, textureList *object.TextureList, Width, Height int64) object.Object {
	if texture, ok := textureList.Get(node.Name.Value); ok {
		if trans, ok := env.Get(node.Transform.Value); ok {
			go transform.TransformEval(trans.(*object.Transform).Body, texture, env, Width, Height)
		} else {
			go transform.TransformEval(screenBuiltins["default"], texture, env, Width, Height)
		}

		LayerMutex.Lock()
		layerList.Layers[0].AddNewTexture(texture)
		LayerMutex.Unlock()

		return nil
	}

	if root, ok := env.Get(node.Name.Value); ok {
		texture, suc := renderer.LoadFromFile(apsolutedRoot + root.(*object.Image).Root.Inspect())
		if !suc {
			return nil
		}

		if trans, ok := env.Get(node.Transform.Value); ok {
			go transform.TransformEval(trans.(*object.Transform).Body, texture, env, Width, Height)
		} else {
			switch node.Transform.Value {
			case "default":
				go transform.TransformEval(screenBuiltins["default"], texture, env, Width, Height)
			}
		}

		LayerMutex.Lock()
		layerList.Layers[0].AddNewTexture(texture)
		LayerMutex.Unlock()

		textureList.Set(node.Name.String(), texture)
	}

	return nil
}
