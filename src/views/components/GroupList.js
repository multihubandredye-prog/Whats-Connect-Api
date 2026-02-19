import GroupListParticipants from "./GroupListParticipants.js";

export default {
    name: 'ListGroup',
    components: { GroupListParticipants },
    props: ['connected'],
    data() {
        return {
            groups: [],
            selectedGroupId: null,
            requestedMembers: [],
            loadingRequestedMembers: false,
            processingMember: null,
        }
    },
    computed: {
        currentUserId() {
            // connected can be array of objects with device/id or may be undefined
            if (!this.connected || this.connected.length === 0) return null;
            const entry = this.connected[0];
            const raw = entry.device || entry.id || '';
            if (!raw || typeof raw !== 'string') return null;
            return raw.split('@')[0].split(':')[0];
        }
    },
    methods: {
        async openModal() {
            try {
                this.dtClear();
                // Reset groups before fetching new data
                this.groups = [];
                await this.submitApi();
                $('#modalGroupList').modal('show');
                // Wait a bit for modal animation to complete before initializing DataTable
                await new Promise(resolve => setTimeout(resolve, 100));
                await this.dtRebuild();
                showSuccessInfo("Grupos carregados")
            } catch (err) {
                showErrorInfo(err)
            }
        },
        dtClear() {
            const table = $('#account_groups_table');
            if ($.fn.DataTable.isDataTable(table)) {
                table.DataTable().clear().destroy();
            }
        },
        async dtRebuild() {
            // Wait for Vue to render the new data
            await this.$nextTick();
            // Additional delay to ensure DOM is fully updated after Vue render
            await new Promise(resolve => setTimeout(resolve, 100));
            const table = $('#account_groups_table');
            if ($.fn.DataTable.isDataTable(table)) {
                table.DataTable().destroy();
            }
            table.DataTable({
                pageLength: 100,
            });
        },
        async handleLeaveGroup(group_id) {
            try {
                const ok = confirm("Tem certeza que deseja sair deste grupo?");
                if (!ok) return;

                await this.leaveGroupApi(group_id);
                this.dtClear()
                await this.submitApi();
                this.dtRebuild()
                showSuccessInfo("Grupo saiu")
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async leaveGroupApi(group_id) {
            try {
                let payload = new FormData();
                payload.append("group_id", group_id)
                await window.http.post(`/group/leave`, payload)
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);

            }
        },
        async submitApi() {
            try {
                let response = await window.http.get(`/user/my/groups`)
                // Ensure groups is always an array, even if null/undefined
                this.groups = response.data.results.data || [];
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            }
        },
        formatDate: function (value) {
            if (!value) return ''
            return moment(value).format('LLL');
        },
        isAdmin(group) {
            // Check if current user is the owner
            const owner = group.OwnerJID.split('@')[0];
            if (owner === this.currentUserId) {
                return true;
            }
            
            // Check if current user is an admin in participants
            const currentUserJID = `${this.currentUserId}@s.whatsapp.net`;
            const participant = group.Participants.find(p => p.PhoneNumber === currentUserJID);
            return participant && participant.IsAdmin;
        },
        async handleSeeRequestedMember(group_id) {
            this.selectedGroupId = group_id;
            this.loadingRequestedMembers = true;
            this.requestedMembers = [];
            
            try {
                const response = await window.http.get(`/group/participant-requests?group_id=${group_id}`);
                this.requestedMembers = response.data.results || [];
                this.loadingRequestedMembers = false;
                $('#modalRequestedMembers').modal('show');
            } catch (error) {
                this.loadingRequestedMembers = false;
                let errorMessage = "Falha ao buscar membros solicitados";
                if (error.response) {
                    errorMessage = error.response.data.message || errorMessage;
                }
                showErrorInfo(errorMessage);
            }
        },
        async handleSeeParticipants(group) {
            if (!group || !group.JID) return;

            this.selectedGroupId = group.JID;
            $('#modalGroupList').modal('hide');

            try {
                await this.$refs.participantsModal.open(group);
            } catch (error) {
                const errorMessage = error?.message || 'Falha ao buscar participantes';
                showErrorInfo(errorMessage);
                $('#modalGroupList').modal('show');
            }
        },
        handleExportParticipants(group) {
            if (!group || !group.JID) return;

            const baseURL = (window.http && window.http.defaults && window.http.defaults.baseURL) ? window.http.defaults.baseURL : '';
            const exportUrl = `${baseURL}/group/participants/export?group_id=${encodeURIComponent(group.JID)}`;
            window.open(exportUrl, '_blank');
        },
        formatJID(jid) {
            return jid ? jid.split('@')[0] : '';
        },
        closeRequestedMembersModal() {
            $('#modalRequestedMembers').modal('hide');
            // open modal again
            this.openModal();
        },
        handleParticipantsClosed() {
            $('#modalGroupList').modal('show');
        },
        async handleProcessRequest(member, action) {
            if (!this.selectedGroupId || !member) return;

            const actionText = action === 'approve' ? 'aprovar' : 'rejeitar';
            const confirmMsg = `Tem certeza que deseja ${actionText} esta solicitação de membro?`;
            const ok = confirm(confirmMsg);
            if (!ok) return;

            try {
                this.processingMember = member.jid;

                const payload = {
                    group_id: this.selectedGroupId,
                    participants: [this.formatJID(member.jid)]
                };

                await window.http.post(`/group/participant-requests/${action}`, payload);

                // Remove the processed member from the list
                this.requestedMembers = this.requestedMembers.filter(m => m.jid !== member.jid);

                showSuccessInfo(`Solicitação de membro ${actionText}da`);
                this.processingMember = null;
            } catch (error) {
                this.processingMember = null;
                let errorMessage = `Falha ao ${actionText} solicitação de membro`;
                if (error.response) {
                    errorMessage = error.response.data.message || errorMessage;
                }
                showErrorInfo(errorMessage);
            }
        }
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui green right ribbon label">Grupo</a>
            <div class="header">Listar Grupos</div>
            <div class="description">
                Exibir todos os seus grupos
            </div>
        </div>
    </div>
    
    <!--  Modal AccountGroup  -->
    <div class="ui large modal" id="modalGroupList">
        <i class="close icon"></i>
        <div class="header">
            Minha Lista de Grupos
        </div>
        <div class="content">
            <table class="ui celled table" id="account_groups_table">
                <thead>
                <tr>
                    <th>ID do Grupo</th>
                    <th>Nome</th>
                    <th>Participantes</th>
                    <th>Criado Em</th>
                    <th>Ação</th>
                </tr>
                </thead>
                <tbody>
                <tr v-for="g in groups" :key="g.JID">
                    <td>{{ g.JID.split('@')[0] }}</td>
                    <td>{{ g.Name }}</td>
                    <td>{{ g.Participants.length }}</td>
                    <td>{{ formatDate(g.GroupCreated) }}</td>
                    <td>
                        <div style="display: flex; gap: 8px; align-items: center;">
                            <button class="ui blue tiny button" @click="handleSeeParticipants(g)">Participantes</button>
                            <button class="ui grey tiny button" @click="handleExportParticipants(g)">Exportar CSV</button>
                            <button v-if="isAdmin(g)" class="ui green tiny button" @click="handleSeeRequestedMember(g.JID)">Membros Solicitados</button>
                            <button class="ui red tiny button" @click="handleLeaveGroup(g.JID)">Sair</button>
                        </div>
                    </td>
                </tr>
                </tbody>
            </table>
        </div>
    </div>

    <group-list-participants ref="participantsModal" @closed="handleParticipantsClosed"></group-list-participants>

    <!-- Requested Members Modal -->
    <div class="ui modal" id="modalRequestedMembers">
        <i class="close icon"></i>
        <div class="header">
            Membros Solicitados do Grupo
        </div>
        <div class="content">
            <div v-if="loadingRequestedMembers" class="ui active centered inline loader"></div>
            
            <div v-else-if="requestedMembers.length === 0" class="ui info message">
                <div class="header">Nenhum Membro Solicitado</div>
                <p>Não há solicitações de membros pendentes para este grupo.</p>
            </div>
            
            <table v-else class="ui celled table">
                <thead>
                    <tr>
                        <th>Usuário</th>
                        <th>Número de Telefone</th>
                        <th>Hora da Solicitação</th>
                        <th>Ação</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="member in requestedMembers" :key="member.jid">
                        <td>
                            <div class="header">{{ formatJID(member.jid) }}</div>
                            <div v-if="member.display_name" class="description">{{ member.display_name }}</div>
                        </td>
                        <td>{{ member.phone_number || formatJID(member.jid) }}</td>
                        <td>{{ formatDate(member.requested_at) }}</td>
                        <td>
                            <div class="ui mini buttons">
                                <button class="ui green button" 
                                        @click="handleProcessRequest(member, 'approve')"
                                        :disabled="processingMember === member.jid">
                                    <i v-if="processingMember === member.jid" class="spinner loading icon"></i>
                                    Aprovar
                                </button>
                                <div class="or"></div>
                                <button class="ui red button" 
                                        @click="handleProcessRequest(member, 'reject')"
                                        :disabled="processingMember === member.jid">
                                    <i v-if="processingMember === member.jid" class="spinner loading icon"></i>
                                    Rejeitar
                                </button>
                            </div>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
        <div class="actions">
            <div class="ui button" @click="closeRequestedMembersModal">Fechar</div>
        </div>
    </div>
    `
}
