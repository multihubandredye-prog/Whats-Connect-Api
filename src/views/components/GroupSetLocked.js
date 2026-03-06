export default {
    name: 'GroupSetLocked',
    data() {
        return {
            loading: false,
            groupId: '',
            locked: false,
        }
    },
    methods: {
        openModal() {
            $('#modalGroupSetLocked').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            return this.groupId.trim() !== '';
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }
            try {
                let response = await this.submitApi()
                showSuccessInfo(response)
                $('#modalGroupSetLocked').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let response = await window.http.post(`/group/locked`, {
                    group_id: this.groupId,
                    locked: this.locked
                })
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.groupId = '';
            this.locked = false;
        },
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui green right ribbon label">Grupo</a>
            <div class="header">Definir Bloqueio do Grupo</div>
            <div class="description">
                Bloquear/desbloquear edição de informações do grupo apenas para administradores
            </div>
        </div>
    </div>
    
    <!--  Modal Group Set Locked  -->
    <div class="ui small modal" id="modalGroupSetLocked">
        <i class="close icon"></i>
        <div class="header">
            Definir Status de Bloqueio do Grupo
        </div>
        <div class="content">
            <form class="ui form">
                <div class="field">
                    <label>ID do Grupo</label>
                    <input v-model="groupId" type="text"
                           placeholder="120363024512399999@g.us"
                           aria-label="Group ID">
                </div>
                
                <div class="field">
                    <label>Status de Bloqueio</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" v-model="locked">
                        <label>{{ locked ? 'Bloquear grupo (apenas administradores podem editar informações do grupo)' : 'Desbloquear grupo (todos os membros podem editar informações do grupo)' }}</label>
                    </div>
                    <div class="ui info message" style="margin-top: 10px;">
                        <div class="header">O que isso faz?</div>
                        <ul class="list">
                            <li><strong>Bloqueado:</strong> Apenas administradores do grupo podem alterar nome, descrição e foto do grupo</li>
                            <li><strong>Desbloqueado:</strong> Todos os membros do grupo podem alterar informações do grupo</li>
                        </ul>
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                    :class="{'loading': this.loading, 'disabled': !this.isValidForm() || this.loading}"
                    @click.prevent="handleSubmit" type="button">
                {{ locked ? 'Bloquear Grupo' : 'Desbloquear Grupo' }}
                <i :class="locked ? 'lock icon' : 'unlock icon'"></i>
            </button>
        </div>
    </div>
    `
} 