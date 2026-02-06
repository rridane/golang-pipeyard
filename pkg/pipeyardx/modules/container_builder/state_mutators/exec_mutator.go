package state_mutators

import "github.com/moby/buildkit/client/llb"

/// RunOptions typiques
//llb.Shlex("go build ./..."),           // commande
//llb.AddMount("/src", src),             // input supplémentaire
//llb.ReadonlyRootFS(),                  // fs racine en RO
//llb.AddSSHSocket(llb.SSHID("default")),// SSH agent
//llb.AddSecret("/run/secrets/tok",      // secret monté en fichier
//llb.SecretID("my-token"),
//llb.SecretFileOpt(1000,1000,0400)),
//llb.ValidExitCodes(0,2),               // tolérer certains codes
//// Toute StateOption peut aussi être passée ici, à portée "apply only":
//llb.AddEnv("CGO_ENABLED","0"),         // ENV (non persistant)
//llb.User("builder"),                   // user (non persistant)
//llb.Network(llb.NetModeHost),          // réseau (non persistant)
//llb.Security(llb.SecurityModeSandbox), // sécurité (non persistant)

type ExecMutator struct {
	Options []llb.RunOption
}

func (e *ExecMutator) Mutate(st llb.State) llb.State {

	return st.Run(
		e.Options...,
	).Root()

}
